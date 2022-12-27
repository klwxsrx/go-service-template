package message

import (
	"context"
	"errors"
)

type EventTypeHandlerMap map[string]Handler

type EventHandlerComposite struct {
	eventTypeDecoder EventTypeDecoder
	subscribers      map[string][]Handler
}

func (h *EventHandlerComposite) Handle(ctx context.Context, msg *Message) error {
	eventType, err := h.eventTypeDecoder.EventType(msg)
	if errors.Is(err, ErrEventTypeDecoderEventTypeNotFound) {
		return nil
	}
	handlers, ok := h.subscribers[eventType]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		err := handler.Handle(ctx, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *EventHandlerComposite) Subscribe(handlers EventTypeHandlerMap) {
	for eventType, handler := range handlers {
		h.subscribers[eventType] = append(h.subscribers[eventType], handler)
	}
}

func NewEventHandlerComposite(eventTypeDecoder EventTypeDecoder) *EventHandlerComposite {
	return &EventHandlerComposite{
		eventTypeDecoder: eventTypeDecoder,
		subscribers:      make(map[string][]Handler),
	}
}
