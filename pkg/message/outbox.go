package message

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
)

const defaultOutboxBatchSize = 100

type (
	Outbox interface {
		Worker(context.Context) error
		Process()
	}

	OutboxOption func(*OutboxImpl)

	OutboxImpl struct {
		BatchSize        int
		Retry            backoff.BackOff
		OnInternalError  []func(context.Context, error)
		OnFoundMessages  []func(context.Context, []Message, error)
		OnSentMessage    []func(context.Context, *Message, error)
		OnDeletedMessage []func(context.Context, *Message, error)

		storage     Storage
		producer    Producer
		processChan chan struct{}
	}
)

func NewOutbox(
	storage Storage,
	producer Producer,
	opts ...OutboxOption,
) Outbox {
	defaultRetry := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Second),
		backoff.WithMultiplier(2),
		backoff.WithMaxInterval(time.Minute),
		backoff.WithMaxElapsedTime(0),
	)

	o := &OutboxImpl{
		BatchSize:        defaultOutboxBatchSize,
		Retry:            defaultRetry,
		OnInternalError:  nil,
		OnFoundMessages:  nil,
		OnSentMessage:    nil,
		OnDeletedMessage: nil,

		storage:     storage,
		producer:    producer,
		processChan: make(chan struct{}, 1),
	}
	for _, opt := range opts {
		opt(o)
	}

	return o
}

func (o *OutboxImpl) Worker(ctx context.Context) error {
	o.Process()

	for {
		select {
		case <-o.processChan:
			o.process(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (o *OutboxImpl) Process() {
	select {
	case o.processChan <- struct{}{}:
	default:
	}
}

func (o *OutboxImpl) process(ctx context.Context) {
	impl := func() error {
		var err error
		var allProcessed bool
		for !allProcessed {
			allProcessed, err = o.processBatch(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	}

	_ = backoff.Retry(impl, backoff.WithContext(o.Retry, ctx))
}

func (o *OutboxImpl) processBatch(ctx context.Context) (allProcessed bool, err error) {
	ctx, releaseLock, err := o.storage.Lock(ctx)
	if err != nil {
		err = fmt.Errorf("get storage lock: %w", err)
		for _, fn := range o.OnInternalError {
			fn(ctx, err)
		}
		return false, err
	}
	defer func() {
		err = releaseLock()
		if err != nil {
			for _, fn := range o.OnInternalError {
				fn(ctx, fmt.Errorf("release storage lock: %w", err))
			}
		}
	}()

	msgs, err := o.storage.Find(ctx, &StorageSpecification{
		ScheduledAtBefore: time.Now(),
		Limit:             o.BatchSize,
	})
	for _, fn := range o.OnFoundMessages {
		fn(ctx, msgs, err)
	}
	if err != nil {
		err = fmt.Errorf("get messages to send: %w", err)
		return false, err
	}
	if len(msgs) == 0 {
		return true, nil
	}

	for _, msg := range msgs {
		err = o.producer.Produce(ctx, &msg)
		for _, fn := range o.OnSentMessage {
			fn(ctx, &msg, err)
		}
		if err != nil {
			return false, fmt.Errorf("send message: %w", err)
		}

		err = o.storage.Delete(ctx, msg.Topic, msg.ID)
		for _, fn := range o.OnDeletedMessage {
			fn(ctx, &msg, err)
		}
		if err != nil {
			return false, fmt.Errorf("delete sent message: %w", err)
		}
	}

	return len(msgs) < o.BatchSize, nil
}

func WithOutboxRetry(retry backoff.BackOff) OutboxOption {
	return func(o *OutboxImpl) {
		o.Retry = retry
	}
}

func WithOutboxLogging(
	logger log.Logger,
	infoLevel log.Level,
	errorLevel log.Level,
) OutboxOption {
	return func(o *OutboxImpl) {
		o.OnInternalError = append(o.OnInternalError, func(ctx context.Context, err error) {
			logger.WithError(err).Log(ctx, errorLevel, "message outbox internal error")
		})

		o.OnFoundMessages = append(o.OnFoundMessages, func(ctx context.Context, _ []Message, err error) {
			if err != nil {
				logger.WithError(err).Log(ctx, errorLevel, "message outbox internal error")
			}
		})

		o.OnSentMessage = append(o.OnSentMessage, func(ctx context.Context, msg *Message, err error) {
			logger := logger.WithField("messageID", msg.ID)
			if err != nil {
				logger.WithError(err).Log(ctx, errorLevel, "outbox message sending failed")
			} else {
				logger.Log(ctx, infoLevel, "outbox message sent successfully")
			}
		})

		o.OnDeletedMessage = append(o.OnDeletedMessage, func(ctx context.Context, msg *Message, err error) {
			logger := logger.WithField("messageID", msg.ID)
			if err != nil {
				err = fmt.Errorf("delete message from storage: %w", err)
				logger.WithError(err).Log(ctx, errorLevel, "message outbox internal error")
			} else {
				logger.Log(ctx, infoLevel, "outbox message deleted from storage successfully")
			}
		})
	}
}

func WithOutboxMetrics(metrics metric.Metrics) OutboxOption {
	return func(o *OutboxImpl) {
		o.OnInternalError = append(o.OnInternalError, func(context.Context, error) {
			metrics.Increment("msg_outbox_internal_error_total")
		})

		o.OnFoundMessages = append(o.OnFoundMessages, func(_ context.Context, _ []Message, err error) {
			if err != nil {
				metrics.Increment("msg_outbox_internal_error_total")
			}
		})

		o.OnSentMessage = append(o.OnSentMessage, func(_ context.Context, msg *Message, err error) {
			metrics.With(metric.Labels{
				"success": err == nil,
				"topic":   msg.Topic,
			}).Increment("msg_outbox_producer_sending_attempts_total")
		})

		o.OnDeletedMessage = append(o.OnDeletedMessage, func(_ context.Context, msg *Message, err error) {
			metrics.With(metric.Labels{
				"success": err == nil,
				"topic":   msg.Topic,
			}).Increment("msg_outbox_producer_delete_from_storage_attempts_total")
		})
	}
}
