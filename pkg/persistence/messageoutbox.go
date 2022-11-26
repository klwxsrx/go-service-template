package persistence

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	pkgsync "github.com/klwxsrx/go-service-template/pkg/sync"
	"sync"
	"time"
)

type MessageOutbox interface {
	Process()
	Close()
}

type messageOutbox struct {
	store      MessageStore
	senderImpl message.Sender
	critical   pkgsync.CriticalSection
	retry      *backoff.ExponentialBackOff
	logger     log.Logger

	processChan chan struct{}
	stopChan    chan struct{}
	onceCloser  *sync.Once
}

func (o *messageOutbox) Process() {
	select {
	case o.processChan <- struct{}{}:
	default:
	}
}

func (o *messageOutbox) Close() {
	o.onceCloser.Do(func() {
		o.stopChan <- struct{}{}
	})
}

func (o *messageOutbox) run() {
	for {
		select {
		case <-o.processChan:
			o.processSend()
		case <-o.stopChan:
			return
		}
	}
}

func (o *messageOutbox) processSend() {
	ctx := context.Background()

	process := func() error {
		var atLeastOneProcessed bool
		var err error
		err = o.critical.Execute(ctx, "process_message_dispatch", func() error {
			for atLeastOneProcessed && err == nil {
				atLeastOneProcessed, err = o.processSendBatch(ctx)
			}
			return err
		})
		return err
	}

	_ = backoff.Retry(func() error {
		err := process()
		if err != nil {
			o.logger.WithError(err).Error(ctx, "failed to get process messages")
		}
		return err
	}, o.retry)
}

func (o *messageOutbox) processSendBatch(ctx context.Context) (atLeastOneProcessed bool, err error) {
	msgs, err := o.store.GetBatch(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get messages for send: %w", err)
	}
	if len(msgs) == 0 {
		return false, nil
	}

	for _, msg := range msgs {
		v := msg
		err := o.senderImpl.Send(ctx, &v)
		if err != nil {
			return false, fmt.Errorf("failed to send message: %w", err)
		}

		err = o.store.Delete(ctx, []uuid.UUID{msg.ID})
		if err != nil {
			return false, fmt.Errorf("failed to delete sent messages: %w", err)
		}
	}
	return true, nil
}

func NewMessageOutbox(
	out message.Sender,
	store MessageStore,
	critical pkgsync.CriticalSection,
	logger log.Logger,
) MessageOutbox {
	retry := backoff.NewExponentialBackOff()
	retry.InitialInterval = time.Second
	retry.RandomizationFactor = 0
	retry.MaxInterval = time.Minute
	retry.Multiplier = 2
	retry.MaxElapsedTime = 0

	mo := &messageOutbox{
		store:       store,
		senderImpl:  out,
		critical:    critical,
		retry:       retry,
		logger:      logger,
		processChan: make(chan struct{}, 1),
		stopChan:    make(chan struct{}),
		onceCloser:  &sync.Once{},
	}
	go mo.run()
	return mo
}
