package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type eventHandler[T event.Event] struct {
	serializer EventSerializerTyped[T]
	handlers   []event.Handler[T]
}

func (h *eventHandler[T]) Handle(ctx context.Context, msg *Message) error {
	evt, err := h.serializer.Deserialize(msg)
	if errors.Is(err, ErrEventSerializerUnknownEventType) {
		return ErrHandlerUnknownEvent
	}
	if err != nil {
		return fmt.Errorf("failed to deserialize message: %w", err)
	}

	for _, handler := range h.handlers {
		err := handler(ctx, evt)
		if err != nil {
			return fmt.Errorf("failed to handle event: %w", err)
		}
	}
	return nil
}

func NewEventHandler[T event.Event](
	serializer EventSerializerTyped[T],
	handlers ...event.Handler[T],
) Handler {
	return &eventHandler[T]{
		serializer: serializer,
		handlers:   handlers,
	}
}
