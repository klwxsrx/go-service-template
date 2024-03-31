package message

import (
	"context"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type (
	Message struct {
		ID    uuid.UUID
		Topic Topic
		// Key is used for topic partitioning, messages with the same key will fall in the same topic partition
		Key     string
		Payload []byte
	}

	StructuredMessage interface {
		ID() uuid.UUID
		Type() string
	}

	TypedHandler[M any] func(ctx context.Context, msg M) error
)

func NewCompositeHandler[M any](handlers []TypedHandler[M], optionalPool worker.Pool) TypedHandler[M] {
	if len(handlers) == 0 {
		return func(ctx context.Context, msg M) error {
			return nil
		}
	}

	if len(handlers) == 1 {
		return handlers[0]
	}

	if optionalPool == nil {
		return func(ctx context.Context, msg M) error {
			for _, handler := range handlers {
				err := handler(ctx, msg)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}

	return func(ctx context.Context, msg M) error {
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
