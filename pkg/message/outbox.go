package message

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

const (
	DefaultOutboxProcessingInterval = time.Second

	messagesBatchSize = 100
)

type OutboxStorage interface {
	Lock(ctx context.Context) (release func() error, err error)
	GetBatch(ctx context.Context, scheduledBefore time.Time, batchSize int) ([]Message, error)
	Store(ctx context.Context, msgs []Message, scheduledAt time.Time) error
	Delete(ctx context.Context, ids []uuid.UUID) error
}

type Outbox interface {
	Process()
	Close()
}

type outbox struct {
	storage     OutboxStorage
	out         Producer
	transaction persistence.Transaction
	retry       *backoff.ExponentialBackOff
	logger      log.Logger

	processChan chan struct{}
	stopChan    chan struct{}
	onceCloser  *sync.Once
}

func NewOutbox(
	storage OutboxStorage,
	out Producer,
	transaction persistence.Transaction,
	logger log.Logger,
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
		transaction: transaction,
		retry:       retry,
		logger:      logger,
		processChan: make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		onceCloser:  &sync.Once{},
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
			o.logger.WithError(err).Error(ctx, "failed to get process messages")
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
			o.logger.WithError(err).Error(ctx, "failed to release message outbox storage lock")
		}
	}()

	msgs, err := o.storage.GetBatch(ctx, time.Now(), messagesBatchSize)
	if err != nil {
		return false, fmt.Errorf("get messages for send: %w", err)
	}
	if len(msgs) == 0 {
		return false, nil
	}

	for _, msg := range msgs {
		msg := msg

		err = o.out.Produce(ctx, &msg) // TODO: WithLogging, WithMetrics
		if err != nil {
			return false, fmt.Errorf("send message: %w", err)
		}

		err = o.storage.Delete(ctx, []uuid.UUID{msg.ID})
		if err != nil {
			return false, fmt.Errorf("delete sent messages: %w", err)
		}
	}

	return len(msgs) < messagesBatchSize, nil
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
			}
		}
	}
}
