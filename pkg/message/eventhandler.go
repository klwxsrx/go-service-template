package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type EventHandler struct {
	serializer  EventSerializer
	subscribers map[string][]event.Handler
}

func (h *EventHandler) Handle(ctx context.Context, msg *Message) error {
	evt, err := h.serializer.Deserialize(msg)
	if errors.Is(err, ErrEventSerializerMessageIsNotValidEvent) {
		return nil
	}
	if errors.Is(err, ErrEventSerializerUnknownEventType) {
		return fmt.Errorf("unknown event type: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to deserialize message to event: %w", err)
	}

	handlers, ok := h.subscribers[evt.Type()]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		err := handler.Handle(ctx, evt)
		if err != nil {
			return fmt.Errorf("failed to handle event: %w", err)
		}
	}

	return nil
}

func (h *EventHandler) Subscribe(eventType string, handler event.Handler) {
	h.subscribers[eventType] = append(h.subscribers[eventType], handler)
}

func NewEventHandler(serializer EventSerializer) *EventHandler {
	return &EventHandler{
		serializer:  serializer,
		subscribers: make(map[string][]event.Handler),
	}
}
