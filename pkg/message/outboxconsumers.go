package message

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
)

const (
	consumingMessagesBatchSize = 100
	messageIDLogField          = "messageID"

	defaultConsumerProcessingInterval = time.Second
	defaultMessageConsumingTimeout    = time.Minute
)

type (
	OutboxConsumerProvider struct {
		processingInterval  time.Duration
		msgConsumingTimeout time.Duration
		storage             OutboxStorage
		topicConsumers      map[Topic]*outboxConsumer
		metrics             metric.Metrics
		logger              log.Logger
		loggerInfoLevel     log.Level
		loggerErrorLevel    log.Level
	}

	OutboxConsumerOption func(*OutboxConsumerProvider)
)

func WithOutboxConsumerLogging(logger log.Logger, infoLevel, errorLevel log.Level) OutboxConsumerOption {
	return func(consumers *OutboxConsumerProvider) {
		consumers.logger = logger
		consumers.loggerInfoLevel = infoLevel
		consumers.loggerErrorLevel = errorLevel
	}
}

func WithOutboxConsumerMetrics(metrics metric.Metrics) OutboxConsumerOption {
	return func(consumers *OutboxConsumerProvider) {
		consumers.metrics = metrics
	}
}

func WithOutboxConsumerProcessingInterval(interval time.Duration) OutboxConsumerOption {
	return func(consumers *OutboxConsumerProvider) {
		consumers.processingInterval = interval
	}
}

func WithOutboxConsumerMessageTimeout(timeout time.Duration) OutboxConsumerOption {
	return func(consumers *OutboxConsumerProvider) {
		consumers.msgConsumingTimeout = timeout
	}
}

func NewOutboxConsumerProvider(storage OutboxStorage, opts ...OutboxConsumerOption) *OutboxConsumerProvider {
	consumers := &OutboxConsumerProvider{
		processingInterval:  defaultConsumerProcessingInterval,
		msgConsumingTimeout: defaultMessageConsumingTimeout,
		storage:             storage,
		topicConsumers:      make(map[Topic]*outboxConsumer),
		metrics:             pkgmetricstub.NewMetrics(),
		logger:              log.New(log.LevelDisabled),
	}

	for _, opt := range opts {
		opt(consumers)
	}

	return consumers
}

func (c *OutboxConsumerProvider) Consumer(topic Topic, _ SubscriberName, _ ConsumptionType) (Consumer, error) {
	consumer, ok := c.topicConsumers[topic]
	if ok {
		return consumer, nil
	}

	consumer = newOutboxConsumer(
		topic,
		c.storage,
		c.msgConsumingTimeout,
		c.metrics,
		c.logger,
		c.loggerInfoLevel,
		c.loggerErrorLevel,
	)
	c.topicConsumers[topic] = consumer

	return consumer, nil
}

func (c *OutboxConsumerProvider) Run(ctx context.Context) error {
	for _, consumer := range c.topicConsumers {
		consumer.Run(ctx, c.processingInterval)
	}

	<-ctx.Done()
	return nil
}

type outboxConsumer struct {
	topic               Topic
	storage             OutboxStorage
	msgConsumingTimeout time.Duration
	metrics             metric.Metrics
	logger              log.Logger
	loggerInfoLevel     log.Level
	loggerErrorLevel    log.Level
	onceRunner          *sync.Once
	retry               *backoff.ExponentialBackOff
	closeMutex          *sync.Mutex
	consumingCond       *sync.Cond
	consumingMessages   map[uuid.UUID]struct{}
	consumerCh          chan *ConsumerMessage
	isClosed            bool
}

func newOutboxConsumer(
	topic Topic,
	storage OutboxStorage,
	messageConsumingTimeout time.Duration,
	metrics metric.Metrics,
	logger log.Logger,
	loggerInfoLevel log.Level,
	loggerErrorLevel log.Level,
) *outboxConsumer {
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = time.Second
	retry.RandomizationFactor = 0
	retry.MaxInterval = time.Minute
	retry.Multiplier = 2
	retry.MaxElapsedTime = 0

	return &outboxConsumer{
		topic:               topic,
		storage:             storage,
		metrics:             metrics,
		logger:              logger,
		loggerInfoLevel:     loggerInfoLevel,
		loggerErrorLevel:    loggerErrorLevel,
		msgConsumingTimeout: messageConsumingTimeout,
		onceRunner:          &sync.Once{},
		retry:               retry,
		closeMutex:          &sync.Mutex{},
		consumingCond:       sync.NewCond(&sync.Mutex{}),
		consumingMessages:   make(map[uuid.UUID]struct{}, consumingMessagesBatchSize),
		consumerCh:          make(chan *ConsumerMessage),
		isClosed:            false,
	}
}

func (c *outboxConsumer) Run(ctx context.Context, processingInterval time.Duration) {
	c.onceRunner.Do(func() {
		go func() {
			ticker := time.NewTicker(processingInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					c.processOutboxMessages(ctx)
					if c.isClosed {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	})
}

func (c *outboxConsumer) Name() string {
	return fmt.Sprintf("outbox/%s", c.topic)
}

func (c *outboxConsumer) Messages() <-chan *ConsumerMessage {
	return c.consumerCh
}

func (c *outboxConsumer) Ack(msg *ConsumerMessage) {
	metrics := c.metrics.WithLabel("topic", msg.Message.Topic)
	metrics.Increment("msg_outbox_consumers_acks_total")

	c.logger.
		WithField("messageID", msg.Message.ID).
		Log(msg.Context, c.loggerInfoLevel, "outbox message acked")

	err := c.storage.Delete(msg.Context, []uuid.UUID{msg.Message.ID})
	if err != nil {
		metrics.Increment("msg_outbox_consumers_storage_delete_errors_total")

		c.logger.
			WithError(err).
			WithField("messageID", msg.Message.ID).
			Log(msg.Context, c.loggerErrorLevel, "failed to delete acked message")
	}

	c.releaseConsumingMessage(msg.Context, msg.Message.ID, "")
}

func (c *outboxConsumer) Nack(msg *ConsumerMessage) {
	c.releaseConsumingMessage(msg.Context, msg.Message.ID, "")

	c.metrics.
		WithLabel("topic", msg.Message.Topic).
		Increment("msg_outbox_consumers_nacks_total")

	c.logger.
		WithField("messageID", msg.Message.ID).
		Log(msg.Context, c.loggerErrorLevel, "outbox message nacked")

	// do nothing
}

func (c *outboxConsumer) Close() {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	c.isClosed = true
	close(c.consumerCh)
}

func (c *outboxConsumer) processOutboxMessages(ctx context.Context) {
	processBatch := func() error {
		var err error
		var allProcessed bool
		for !allProcessed {
			allProcessed, err = c.processOutboxMessagesImpl(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				c.logger.
					WithError(err).
					WithField("topic", c.topic).
					Log(ctx, c.loggerErrorLevel, "failed to process stored outbox messages to consume")
			}
			if err != nil {
				return err
			}
		}

		return nil
	}

	_ = backoff.Retry(processBatch, backoff.WithContext(c.retry, ctx))
}

func (c *outboxConsumer) processOutboxMessagesImpl(ctx context.Context) (allProcessed bool, err error) {
	ctx, releaseLock, err := c.storage.Lock(ctx, "topic", string(c.topic))
	if err != nil {
		return false, fmt.Errorf("get lock for consumer: %w", err)
	}
	defer func() {
		releaseErr := releaseLock()
		if releaseErr != nil && err == nil {
			err = fmt.Errorf("release lock for consumer: %w", err)
		}
	}()

	var msgs []Message
	msgs, err = c.storage.GetBatch(ctx, time.Now(), consumingMessagesBatchSize, string(c.topic))
	if err != nil {
		return false, fmt.Errorf("get message by topic: %w", err)
	}
	if len(msgs) == 0 {
		return true, nil
	}

	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()
	if c.isClosed {
		return true, nil
	}

	defer func() {
		c.consumingCond.L.Lock()
		defer c.consumingCond.L.Unlock()

		for len(c.consumingMessages) > 0 {
			c.consumingCond.Wait()
		}
	}()

	for _, msg := range msgs {
		select {
		case c.consumerCh <- &ConsumerMessage{Context: ctx, Message: msg}:
			c.logger.WithField(messageIDLogField, msg.ID).Log(ctx, c.loggerInfoLevel, "outbox message sent to handler")
			c.addConsumingMessage(msg.ID)
			go func() {
				<-time.After(c.msgConsumingTimeout)
				c.releaseConsumingMessage(ctx, msg.ID, "outbox message got handler timeout")
			}()
		case <-ctx.Done():
			return true, nil
		}
	}

	return false, nil
}

func (c *outboxConsumer) addConsumingMessage(msgID uuid.UUID) {
	c.consumingCond.L.Lock()
	defer c.consumingCond.L.Unlock()

	c.consumingMessages[msgID] = struct{}{}
}

func (c *outboxConsumer) releaseConsumingMessage(ctx context.Context, msgID uuid.UUID, logMsg string) {
	c.consumingCond.L.Lock()
	defer c.consumingCond.L.Unlock()

	if _, ok := c.consumingMessages[msgID]; ok && logMsg != "" {
		c.logger.WithField(messageIDLogField, msgID).Log(ctx, c.loggerInfoLevel, logMsg)
	}

	delete(c.consumingMessages, msgID)
	if len(c.consumingMessages) == 0 {
		c.consumingCond.Signal()
	}
}
