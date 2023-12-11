package message

import (
	"context"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type Message struct {
	ID    uuid.UUID
	Topic string
	// Key is used for topic partitioning, messages with the same key will fall in the same topic partition
	Key     string
	Payload []byte
}

type Handler func(ctx context.Context, msg *Message) error

func NewCompositeHandler(handlers []Handler, optionalPool worker.Pool) Handler {
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

	if optionalPool == nil {
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

	return func(ctx context.Context, msg *Message) error {
		group := worker.WithinFailSafeGroup(ctx, optionalPool)
		for _, handler := range handlers {
			handlerImpl := handler
			group.Do(func(ctx context.Context) error {
				return handlerImpl(ctx, msg)
			})
		}
		return group.Wait()
	}
}
