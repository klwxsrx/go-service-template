package message

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
)

const (
	producingMessagesBatchSize               = 100
	metricNameMessageSendAttemptsTotal       = "msg_outbox_producer_send_attempts_total"
	metricNameMessageDeleteSentAttemptsTotal = "msg_outbox_producer_delete_sent_attempts_total"
)

type (
	OutboxStorage interface {
		Lock(ctx context.Context, extraKeys ...string) (newCtx context.Context, release func() error, err error)
		GetBatch(ctx context.Context, scheduledBefore time.Time, batchSize int, specificTopics ...string) ([]Message, error)
		Store(ctx context.Context, msgs []Message, scheduledAt time.Time) error
		Delete(ctx context.Context, ids []uuid.UUID) error
	}

	OutboxProducer interface {
		Process(context.Context)
		Close()
	}

	OutboxProducerOption func(*outboxProducer)
)

type outboxProducer struct {
	storage          OutboxStorage
	out              Producer
	retry            *backoff.ExponentialBackOff
	metrics          metric.Metrics
	logger           log.Logger
	loggerInfoLevel  log.Level
	loggerErrorLevel log.Level

	processChan chan struct{}
	stopChan    chan struct{}
	onceCloser  *sync.Once
}

func NewOutboxProducer(
	storage OutboxStorage,
	out Producer,
	opts ...OutboxProducerOption,
) OutboxProducer {
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = time.Second
	retry.RandomizationFactor = 0
	retry.MaxInterval = time.Minute
	retry.Multiplier = 2
	retry.MaxElapsedTime = 0

	mo := &outboxProducer{
		storage:     storage,
		out:         out,
		retry:       retry,
		metrics:     metric.NewMetricsStub(),
		logger:      log.New(log.LevelDisabled),
		processChan: make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		onceCloser:  &sync.Once{},
	}

	for _, opt := range opts {
		opt(mo)
	}

	go mo.run()
	mo.Process(context.Background())
	return mo
}

func (o *outboxProducer) Process(_ context.Context) {
	select {
	case o.processChan <- struct{}{}:
	default:
	}
}

func (o *outboxProducer) Close() {
	o.onceCloser.Do(func() {
		o.stopChan <- struct{}{}
	})
}

func (o *outboxProducer) run() {
	for {
		select {
		case <-o.processChan:
			o.processSend()
		case <-o.stopChan:
			return
		}
	}
}

func (o *outboxProducer) processSend() {
	ctx := context.Background()

	process := func() error {
		var err error
		var allProcessed bool
		for !allProcessed {
			allProcessed, err = o.processSendBatch(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	}

	_ = backoff.Retry(func() error {
		err := process()
		if err != nil {
			o.logger.WithError(err).Log(ctx, o.loggerErrorLevel, "failed to process stored outbox messages")
		}
		return err
	}, o.retry)
}

func (o *outboxProducer) processSendBatch(ctx context.Context) (allProcessed bool, err error) {
	ctx, releaseLock, err := o.storage.Lock(ctx)
	if err != nil {
		return false, fmt.Errorf("get storage lock: %w", err)
	}
	defer func() {
		err = releaseLock()
		if err != nil {
			o.logger.WithError(err).Log(ctx, o.loggerErrorLevel, "failed to release message outbox storage lock")
		}
	}()

	msgs, err := o.storage.GetBatch(ctx, time.Now(), producingMessagesBatchSize)
	if err != nil {
		return false, fmt.Errorf("get messages for send: %w", err)
	}
	if len(msgs) == 0 {
		return true, nil
	}

	sentMessages := make([]uuid.UUID, 0, len(msgs))
	defer func() {
		o.logger.WithField("messageIDs", sentMessages).Log(ctx, o.loggerInfoLevel, "outbox messages successfully sent")
	}()

	for _, msg := range msgs {
		metrics := o.metrics.WithLabel("topic", msg.Topic)

		err = o.out.Produce(ctx, &msg)
		if err != nil {
			metrics.WithLabel("success", false).Increment(metricNameMessageSendAttemptsTotal)
			return false, fmt.Errorf("send message: %w", err)
		}
		metrics.WithLabel("success", true).Increment(metricNameMessageSendAttemptsTotal)

		err = o.storage.Delete(ctx, []uuid.UUID{msg.ID})
		if err != nil {
			metrics.WithLabel("success", false).Increment(metricNameMessageDeleteSentAttemptsTotal)
			return false, fmt.Errorf("delete sent messages: %w", err)
		}
		metrics.WithLabel("success", true).Increment(metricNameMessageDeleteSentAttemptsTotal)

		sentMessages = append(sentMessages, msg.ID)
	}

	return len(msgs) < producingMessagesBatchSize, nil
}

func WithOutboxProducerLogging(logger log.Logger, infoLevel log.Level, errorLevel log.Level) OutboxProducerOption {
	return func(outbox *outboxProducer) {
		outbox.logger = logger
		outbox.loggerInfoLevel = infoLevel
		outbox.loggerErrorLevel = errorLevel
	}
}

func WithOutboxProducerMetrics(metrics metric.Metrics) OutboxProducerOption {
	return func(o *outboxProducer) {
		o.metrics = metrics
	}
}
