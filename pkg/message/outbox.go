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

	processMessageOutboxLockName = "process_message_outbox"
)

type OutboxStorage interface {
	GetBatch(ctx context.Context, scheduledBefore time.Time) ([]Message, error)
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
		var atLeastOneProcessed bool
		var err error

		for {
			err = o.transaction.Execute(ctx, func(ctx context.Context) error {
				atLeastOneProcessed, err = o.processSendBatch(ctx)
				return err
			}, processMessageOutboxLockName)
			if !atLeastOneProcessed || err != nil {
				return err
			}
		}
	}

	_ = backoff.Retry(func() error {
		err := process()
		if err != nil {
			o.logger.WithError(err).Error(ctx, "failed to get process messages")
		}
		return err
	}, o.retry)
}

func (o *outbox) processSendBatch(ctx context.Context) (atLeastOneProcessed bool, err error) {
	msgs, err := o.storage.GetBatch(ctx, time.Now())
	if err != nil {
		return false, fmt.Errorf("failed to get messages for send: %w", err)
	}
	if len(msgs) == 0 {
		return false, nil
	}

	for _, msg := range msgs {
		v := msg

		err := o.out.Produce(ctx, &v) // TODO: WithLogging
		if err != nil {
			return false, fmt.Errorf("failed to send message: %w", err)
		}

		err = o.storage.Delete(ctx, []uuid.UUID{msg.ID})
		if err != nil {
			return false, fmt.Errorf("failed to delete sent messages: %w", err)
		}
	}
	return true, nil
}

type outboxProcessor struct {
	ticker *time.Ticker
	outbox Outbox
}

func (o outboxProcessor) Name() string {
	return "message outbox processor"
}

func (o outboxProcessor) Process() worker.Process {

	return func(stopChan <-chan struct{}) error {
		for {
			select {
			case <-o.ticker.C:
				o.outbox.Process()
			case <-stopChan:
				o.ticker.Stop()
				o.outbox.Close()
			}
		}
	}
}

func NewOutboxProcessor(
	processingInterval time.Duration,
	outbox Outbox,
) worker.NamedProcess {
	return outboxProcessor{
		ticker: time.NewTicker(processingInterval),
		outbox: outbox,
	}
}
