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
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

const (
	DefaultOutboxProcessingInterval = time.Second

	messagesBatchSize                       = 100
	metricNameMessageSendAttemptsTotal      = "msg_outbox_send_attempts_total"
	metricNameMessageDeleteSentAttemptTotal = "msg_outbox_delete_sent_attempts_total"
)

type (
	OutboxStorage interface {
		Lock(ctx context.Context) (release func() error, err error)
		GetBatch(ctx context.Context, scheduledBefore time.Time, batchSize int) ([]Message, error)
		Store(ctx context.Context, msgs []Message, scheduledAt time.Time) error
		Delete(ctx context.Context, ids []uuid.UUID) error
	}

	Outbox interface {
		Process()
		Close()
	}

	OutboxOption func(*outbox)
)

type outbox struct {
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

func NewOutbox(
	storage OutboxStorage,
	out Producer,
	opts ...OutboxOption,
) Outbox {
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = time.Second
	retry.RandomizationFactor = 0
	retry.MaxInterval = time.Minute
	retry.Multiplier = 2
	retry.MaxElapsedTime = 0

	mo := &outbox{
		storage:     storage,
		out:         out,
		retry:       retry,
		metrics:     pkgmetricstub.NewMetrics(),
		logger:      log.New(log.LevelDisabled),
		processChan: make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		onceCloser:  &sync.Once{},
	}

	for _, opt := range opts {
		opt(mo)
	}

	go mo.run()
	mo.Process()
	return mo
}

func (o *outbox) Process() {
	select {
	case o.processChan <- struct{}{}:
	default:
	}
}

func (o *outbox) Close() {
	o.onceCloser.Do(func() {
		o.stopChan <- struct{}{}
	})
}

func (o *outbox) run() {
	for {
		select {
		case <-o.processChan:
			o.processSend()
		case <-o.stopChan:
			return
		}
	}
}

func (o *outbox) processSend() {
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
			o.logger.WithError(err).Log(ctx, o.loggerErrorLevel, "failed to get process messages")
		}
		return err
	}, o.retry)
}

func (o *outbox) processSendBatch(ctx context.Context) (allProcessed bool, err error) {
	releaseLock, err := o.storage.Lock(ctx)
	if err != nil {
		return false, fmt.Errorf("get storage lock: %w", err)
	}
	defer func() {
		err := releaseLock()
		if err != nil {
			o.logger.WithError(err).Log(ctx, o.loggerErrorLevel, "failed to release message outbox storage lock")
		}
	}()

	msgs, err := o.storage.GetBatch(ctx, time.Now(), messagesBatchSize)
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
		msg := msg
		metrics := o.metrics.WithLabel("topic", msg.Topic)

		err = o.out.Produce(ctx, &msg)
		if err != nil {
			metrics.WithLabel("success", false).Increment(metricNameMessageSendAttemptsTotal)
			return false, fmt.Errorf("send message: %w", err)
		}
		metrics.WithLabel("success", true).Increment(metricNameMessageSendAttemptsTotal)

		err = o.storage.Delete(ctx, []uuid.UUID{msg.ID})
		if err != nil {
			metrics.WithLabel("success", false).Increment(metricNameMessageDeleteSentAttemptTotal)
			return false, fmt.Errorf("delete sent messages: %w", err)
		}
		metrics.WithLabel("success", true).Increment(metricNameMessageDeleteSentAttemptTotal)

		sentMessages = append(sentMessages, msg.ID)
	}

	return len(msgs) < messagesBatchSize, nil
}

func WithOutboxLogging(logger log.Logger, infoLevel log.Level, errorLevel log.Level) OutboxOption {
	return func(outbox *outbox) {
		outbox.logger = logger
		outbox.loggerInfoLevel = infoLevel
		outbox.loggerErrorLevel = errorLevel
	}
}

func WithOutboxMetrics(metrics metric.Metrics) OutboxOption {
	return func(o *outbox) {
		o.metrics = metrics
	}
}

func NewOutboxProcessor(
	outbox Outbox,
	processingInterval time.Duration,
) worker.Process {
	ticker := time.NewTicker(processingInterval)
	return func(ctx context.Context) error {
		for {
			select {
			case <-ticker.C:
				outbox.Process()
			case <-ctx.Done():
				ticker.Stop()
				outbox.Close()
				return nil
			}
		}
	}
}
