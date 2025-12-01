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

	DispatcherImpl struct {
		handlers map[string][]Handler
	}
)

func NewDispatcher(handlers ...RegisterHandlerFunc) (*DispatcherImpl, error) {
	d := DispatcherImpl{handlers: make(map[string][]Handler)}
	if err := d.Register(handlers...); err != nil {
		return nil, err
	}

	return &d, nil
}

func (d *DispatcherImpl) Dispatch(ctx context.Context, events ...Event) error {
	for _, evt := range events {
		handlers, ok := d.handlers[evt.Type()]
		if !ok {
			return fmt.Errorf("handler not registered for %s", evt.Type())
		}

		for _, handler := range handlers {
			if err := handler(ctx, evt); err != nil {
				return fmt.Errorf("handle event by %T: %w", handler, err)
			}
		}
	}

	return nil
}

func (d *DispatcherImpl) Register(handlers ...RegisterHandlerFunc) error {
	if len(handlers) == 0 {
		return nil
	}

	for _, registerFunc := range handlers {
		eventType, handler, err := registerFunc()
		if err != nil {
			return err
		}

		d.handlers[eventType] = append(d.handlers[eventType], handler)
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
