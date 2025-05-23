package event

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type (
	Event interface {
		ID() uuid.UUID
		Type() string
		AggregateID() uuid.UUID
	}

	Dispatcher interface {
		Dispatch(ctx context.Context, events ...Event) error
	}

	TypedHandler[T Event] func(ctx context.Context, event T) error
	Handler               TypedHandler[Event]

	RegisterHandlerFunc func() (eventType string, handler Handler, err error)

	dispatcher struct {
		handlers map[string]Handler
	}
)

func NewDispatcher(handler RegisterHandlerFunc, handlers ...RegisterHandlerFunc) (Dispatcher, error) {
	handlers = append([]RegisterHandlerFunc{handler}, handlers...)
	handlersMap := make(map[string]Handler, len(handlers))
	for _, registerFunc := range handlers {
		eventType, handler, err := registerFunc()
		if err != nil {
			return nil, err
		}
		if _, ok := handlersMap[eventType]; ok {
			return nil, fmt.Errorf("event handler for %s already exists", eventType)
		}

		handlersMap[eventType] = handler
	}

	return dispatcher{handlers: handlersMap}, nil
}

func (d dispatcher) Dispatch(ctx context.Context, events ...Event) error {
	for _, evt := range events {
		handler, ok := d.handlers[evt.Type()]
		if !ok {
			return fmt.Errorf("handler not registered for %s", evt.Type())
		}

		err := handler(ctx, evt)
		if err != nil {
			return fmt.Errorf("handle event: %w", err)
		}
	}
	return nil
}

func RegisterHandler[T Event](handler TypedHandler[T]) RegisterHandlerFunc {
	return func() (string, Handler, error) {
		var blankEvent T
		eventType := blankEvent.Type()
		if eventType == "" {
			return "", nil, fmt.Errorf("get event type for %T: blank event must return const value", blankEvent)
		}

		return eventType, func(ctx context.Context, event Event) error {
			concreteEvent, ok := event.(T)
			if !ok {
				return fmt.Errorf("invalid event struct type %T, expected %T", event, concreteEvent)
			}
			return handler(ctx, concreteEvent)
		}, nil
	}
}
