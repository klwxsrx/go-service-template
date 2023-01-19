package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type EventTypeHandlerMap map[string]event.Handler

type eventHandler struct {
	deserializer EventDeserializer
	handlers     EventTypeHandlerMap
}

func (h *eventHandler) Handle(ctx context.Context, msg *Message) error {
	eventType, err := h.deserializer.ParseType(msg)
	if errors.Is(err, ErrEventDeserializeNotValidEvent) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to parse event type: %w", err)
	}

	handler, ok := h.handlers[eventType]
	if !ok {
		return nil
	}

	evt, err := h.deserializer.Deserialize(msg)
	if err != nil {
		return fmt.Errorf("failed to deserialize event: %w", err)
	}

	err = handler(ctx, evt)
	if err != nil {
		return fmt.Errorf("failed to handle event: %w", err)
	}
	return nil
}

func NewEventHandler(deserializer EventDeserializer, handlers EventTypeHandlerMap) Handler {
	return &eventHandler{
		deserializer: deserializer,
		handlers:     handlers,
	}
}
