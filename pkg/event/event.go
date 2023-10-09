//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Event=Event,Dispatcher=Dispatcher"
package event

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Event interface {
	ID() uuid.UUID
	Type() string
	AggregateID() uuid.UUID
	AggregateName() string
}

type (
	TypedHandler[T Event] func(ctx context.Context, event T) error
	Handler               TypedHandler[Event]

	RegisterHandlerFunc func() (eventType string, handler Handler, err error)
)

type Dispatcher interface {
	Dispatch(ctx context.Context, events []Event) error
}

type dispatcher struct {
	handlers map[string]Handler
}

func (d dispatcher) Dispatch(ctx context.Context, events []Event) error {
	for _, evt := range events {
		handler, ok := d.handlers[evt.Type()]
		if !ok {
			return fmt.Errorf("handler not registered for %s", evt.Type())
		}

		err := handler(ctx, evt)
		if err != nil {
			return fmt.Errorf("failed to handle event: %w", err)
		}
	}
	return nil
}

func RegisterHandler[T Event](handler TypedHandler[T]) RegisterHandlerFunc {
	return func() (string, Handler, error) {
		var blankEvent T
		eventType := blankEvent.Type()
		if eventType == "" {
			return "", nil, fmt.Errorf("failed to get event type for %T: blank event must return const value", blankEvent)
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

func NewDispatcher(handlers ...RegisterHandlerFunc) (Dispatcher, error) {
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
