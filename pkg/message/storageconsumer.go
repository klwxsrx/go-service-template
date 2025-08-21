package message

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

const defaultStorageConsumerBatchSize = 500

type (
	StorageConsumerProvider interface {
		ConsumerProvider[AckStrategy]
		Process()
		Workers() []worker.ErrorJob
	}

	StorageConsumerProviderOption func(*StorageConsumerProviderImpl)

	StorageConsumerProviderImpl struct {
		ConsumingBatchSize      int
		ConsumingRetry          backoff.BackOff
		OnMessageProcessing     []func(context.Context, Topic, *Message)
		OnAcknowledge           []func(context.Context, Topic, *Message, error)
		OnMessageBatchProcessed []func(context.Context, Topic, int, error)

		storage   Storage
		consumers map[Topic]*storageConsumer
	}

	storageConsumer struct {
		topic                   Topic
		consumingBatchSize      int
		storage                 Storage
		messagesCh              chan *ConsumerMessage
		mutex                   *sync.RWMutex
		processChan             chan struct{}
		processingMessages      map[uuid.UUID]struct{}
		retry                   backoff.BackOff
		onMessageProcessing     []func(context.Context, Topic, *Message)
		onAcknowledge           []func(context.Context, Topic, *Message, error)
		onMessageBatchProcessed []func(context.Context, Topic, int, error)
	}
)

func NewStorageConsumerProvider(storage Storage, opts ...StorageConsumerProviderOption) StorageConsumerProvider {
	defaultRetry := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Second),
		backoff.WithMultiplier(2),
		backoff.WithMaxInterval(time.Minute),
		backoff.WithMaxElapsedTime(0),
	)

	provider := &StorageConsumerProviderImpl{
		ConsumingBatchSize:      defaultStorageConsumerBatchSize,
		ConsumingRetry:          defaultRetry,
		OnMessageProcessing:     nil,
		OnAcknowledge:           nil,
		OnMessageBatchProcessed: nil,

		storage:   storage,
		consumers: make(map[Topic]*storageConsumer),
	}

	for _, opt := range opts {
		opt(provider)
	}

	return provider
}

func (p *StorageConsumerProviderImpl) Consumer(topic Topic, _ Subscriber) (Consumer[AckStrategy], error) {
	_, ok := p.consumers[topic]
	if ok {
		return nil, fmt.Errorf("consumer for topic %s already exists, only one is supported at a time", topic)
	}

	consumer := newStorageConsumer(
		topic,
		p.ConsumingBatchSize,
		p.storage,
		p.ConsumingRetry,
		p.OnMessageProcessing,
		p.OnAcknowledge,
		p.OnMessageBatchProcessed,
	)
	p.consumers[topic] = consumer

	return consumer, nil
}

func (p *StorageConsumerProviderImpl) Process() {
	for _, consumer := range p.consumers {
		consumer.Process()
	}
}

func (p *StorageConsumerProviderImpl) Workers() []worker.ErrorJob {
	workers := make([]worker.ErrorJob, 0, len(p.consumers))
	for _, consumer := range p.consumers {
		workers = append(workers, consumer.Worker)
	}

	return workers
}

func newStorageConsumer(
	topic Topic,
	consumingBatchSize int,
	storage Storage,
	retry backoff.BackOff,
	onMessageProcessing []func(context.Context, Topic, *Message),
	onAcknowledge []func(context.Context, Topic, *Message, error),
	onMessageBatchProcessed []func(context.Context, Topic, int, error),
) *storageConsumer {
	return &storageConsumer{
		topic:                   topic,
		consumingBatchSize:      consumingBatchSize,
		storage:                 storage,
		messagesCh:              make(chan *ConsumerMessage),
		mutex:                   &sync.RWMutex{},
		processChan:             make(chan struct{}, 1),
		processingMessages:      make(map[uuid.UUID]struct{}),
		retry:                   retry,
		onMessageProcessing:     onMessageProcessing,
		onAcknowledge:           onAcknowledge,
		onMessageBatchProcessed: onMessageBatchProcessed,
	}
}

func (c *storageConsumer) Process() {
	select {
	case c.processChan <- struct{}{}:
	default:
	}
}

func (c *storageConsumer) Worker(ctx context.Context) error {
	c.Process()

	for {
		select {
		case <-ctx.Done():
			close(c.messagesCh)
			return ctx.Err()
		case <-c.processChan:
			c.consumeStorageMessages(ctx)
		}
	}
}

func (c *storageConsumer) Topic() Topic {
	return c.topic
}

func (c *storageConsumer) Subscriber() Subscriber {
	return "message-storage-consumer"
}

func (c *storageConsumer) Messages() <-chan *ConsumerMessage {
	return c.messagesCh
}

func (c *storageConsumer) Ack(ctx context.Context, msg *ConsumerMessage) error {
	err := c.storage.Delete(ctx, msg.Message.Topic, msg.Message.ID)
	for _, fn := range c.onAcknowledge {
		fn(ctx, c.topic, &msg.Message, err)
	}
	if err != nil {
		return fmt.Errorf("delete message from storage: %w", err)
	}

	c.removeFromProcessing(msg.Message.ID)
	return nil
}

func (c *storageConsumer) Acknowledge() AckStrategy {
	return c
}

func (c *storageConsumer) Close() error {
	return nil
}

func (c *storageConsumer) consumeStorageMessages(ctx context.Context) {
	_ = backoff.Retry(func() error {
		var allProcessed bool
		for !allProcessed {
			var err error
			var processedCount int
			allProcessed, processedCount, err = c.consumeStorageMessagesBatch(ctx)
			if err != nil || processedCount > 0 {
				for _, fn := range c.onMessageBatchProcessed {
					fn(ctx, c.topic, processedCount, err)
				}
			}
			if err != nil {
				return err
			}
		}

		return nil
	}, backoff.WithContext(c.retry, ctx))
}

func (c *storageConsumer) consumeStorageMessagesBatch(ctx context.Context) (allProcessed bool, processedCount int, err error) {
	ctx, releaseLock, err := c.storage.Lock(ctx, "topic", string(c.topic))
	if err != nil {
		return false, 0, fmt.Errorf("get topic lock: %w", err)
	}
	defer func() {
		releaseErr := releaseLock()
		if releaseErr != nil && err == nil {
			err = fmt.Errorf("release topic lock: %w", err)
		}
	}()

	msgs, err := c.storage.Find(ctx, &StorageSpecification{
		IDsExcluded:       c.getProcessingMessageIDs(),
		Topics:            []Topic{c.topic},
		ScheduledAtBefore: time.Now(),
		Limit:             c.consumingBatchSize,
	})
	if err != nil {
		return false, 0, fmt.Errorf("find storage messages: %w", err)
	}
	if len(msgs) == 0 {
		return true, 0, nil
	}

	for _, msg := range msgs {
		select {
		case c.messagesCh <- &ConsumerMessage{Context: ctx, Message: msg}:
			c.addToProcessing(msg.ID)
			for _, fn := range c.onMessageProcessing {
				fn(ctx, c.topic, &msg)
			}
			processedCount++
		case <-ctx.Done():
			return true, processedCount, ctx.Err()
		}
	}

	return len(msgs) < c.consumingBatchSize, processedCount, nil
}

func (c *storageConsumer) addToProcessing(id uuid.UUID) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.processingMessages[id] = struct{}{}
}

func (c *storageConsumer) removeFromProcessing(id uuid.UUID) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.processingMessages, id)
}

func (c *storageConsumer) getProcessingMessageIDs() []uuid.UUID {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return slices.Collect(maps.Keys(c.processingMessages))
}

func WithStorageConsumerLogging(logger log.Logger, infoLevel, errorLevel log.Level) StorageConsumerProviderOption {
	return func(impl *StorageConsumerProviderImpl) {
		impl.OnMessageProcessing = append(impl.OnMessageProcessing, func(ctx context.Context, topic Topic, msg *Message) {
			logger.
				With(log.Fields{
					"topic":     topic,
					"messageID": msg.ID,
				}).
				Log(ctx, infoLevel, "storage message is being processed")
		})

		impl.OnAcknowledge = append(impl.OnAcknowledge, func(ctx context.Context, topic Topic, msg *Message, err error) {
			logger := logger.With(log.Fields{"topic": topic, "messageID": msg.ID})
			if err != nil && errors.Is(err, ctx.Err()) {
				return
			}
			if err != nil {
				logger.WithError(err).Log(ctx, errorLevel, "failed to delete acknowledged message")
				return
			}

			logger.Log(ctx, infoLevel, "acknowledged message deleted from storage")
		})

		impl.OnMessageBatchProcessed = append(impl.OnMessageBatchProcessed, func(ctx context.Context, _ Topic, _ int, err error) {
			if err == nil {
				return
			}
			if err != nil && errors.Is(err, ctx.Err()) {
				return
			}

			logger.WithError(err).Log(ctx, errorLevel, "failed to process storage messages to consume")
		})
	}
}

func WithStorageConsumerMetrics(metrics metric.Metrics) StorageConsumerProviderOption {
	return func(impl *StorageConsumerProviderImpl) {
		impl.OnMessageProcessing = append(impl.OnMessageProcessing, func(_ context.Context, topic Topic, _ *Message) {
			metrics.WithLabel("topic", topic).Increment("msg_storage_consumer_processing_total")
		})

		impl.OnAcknowledge = append(impl.OnAcknowledge, func(_ context.Context, topic Topic, _ *Message, err error) {
			metrics := metrics.WithLabel("topic", topic)
			if err != nil {
				metrics.Increment("msg_storage_consumer_delete_acked_error_total")
				return
			}

			metrics.Increment("msg_storage_consumer_delete_acked_total")
		})

		impl.OnMessageBatchProcessed = append(impl.OnMessageBatchProcessed, func(_ context.Context, topic Topic, _ int, err error) {
			if err == nil {
				return
			}

			metrics.WithLabel("topic", topic).Increment("msg_storage_consumer_internal_errors_total")
		})
	}
}

func WithStorageConsumerBatchSize(size int) StorageConsumerProviderOption {
	return func(impl *StorageConsumerProviderImpl) {
		impl.ConsumingBatchSize = size
	}
}

func WithStorageConsumerRetry(retry backoff.BackOff) StorageConsumerProviderOption {
	return func(impl *StorageConsumerProviderImpl) {
		impl.ConsumingRetry = retry
	}
}
