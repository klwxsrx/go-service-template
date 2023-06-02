package message

import (
	"context"
	"fmt"
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

func NewCompositeHandler(handlers []Handler, optionalWP worker.Pool) Handler {
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

	if optionalWP == nil {
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
		group := worker.WithFailSafeContext(ctx, optionalWP)
		for _, handler := range handlers {
			group.Do(func(ctx context.Context) error {
				return handler(ctx, msg)
			})
		}

		err := group.Wait()
		if err != nil {
			return fmt.Errorf("failed to handle message with worker pool: %w", err)
		}
		return nil
	}
}
