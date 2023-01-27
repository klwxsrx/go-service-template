package event

import (
	"context"
	"fmt"
)

type Handler func(ctx context.Context, event Event) error

func NewTypedHandler[T Event](
	handler func(ctx context.Context, event T) error,
	handlers ...func(ctx context.Context, event T) error,
) Handler {
	handlers = append([]func(ctx context.Context, event T) error{handler}, handlers...)
	return func(ctx context.Context, event Event) error {
		concreteEvent, ok := event.(T)
		if !ok {
			return fmt.Errorf("invalid event with id %v and type %v passed", event.ID(), event.Type())
		}
		for _, handler := range handlers {
			err := handler(ctx, concreteEvent)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
