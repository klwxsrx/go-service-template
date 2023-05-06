package message

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/worker"
	"sync"
)

type Message struct {
	ID    uuid.UUID
	Topic string
	// Key is used for topic partitioning, messages with the same key will fall in the same topic partition
	Key     string
	Payload []byte
}

type Handler func(ctx context.Context, msg *Message) error

func NewCompositeHandler(optionalWorkerPool worker.Pool, handlers []Handler) Handler {
	if len(handlers) == 0 {
		return func(ctx context.Context, msg *Message) error {
			return nil
		}
	}

	if len(handlers) == 1 {
		handler := handlers[0]
		return func(ctx context.Context, msg *Message) error {
			return handler(ctx, msg)
		}
	}

	if optionalWorkerPool == nil {
		return func(ctx context.Context, msg *Message) error {
			for _, handler := range handlers {
				err := handler(ctx, msg)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}

	wg := &sync.WaitGroup{}
	return func(ctx context.Context, msg *Message) error {
		errChan := make(chan error, 1)

		for _, handler := range handlers {
			wg.Add(1)
			handlerImpl := handler
			err := optionalWorkerPool.Do(func() {
				err := handlerImpl(ctx, msg)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
				wg.Done()
			})
			if err != nil {
				wg.Done()
				return fmt.Errorf("failed to handle message with worker pool: %w", err)
			}
		}

		var err error
		select {
		case err = <-errChan:
		default:
		}
		return err
	}
}
