//go:generate mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Dispatcher=Dispatcher"
package event

import (
	"context"
	"fmt"
)

type Dispatcher interface {
	Dispatch(ctx context.Context, events []Event) error
}

type dispatcher struct {
	handlers map[string]Handler
}

func (d *dispatcher) Dispatch(ctx context.Context, events []Event) error {
	for _, evt := range events {
		handler, ok := d.handlers[evt.Type()]
		if !ok {
			continue
		}

		err := handler(ctx, evt)
		if err != nil {
			return fmt.Errorf("failed to handle event: %w", err)
		}
	}
	return nil
}

func NewDispatcher(handlers map[string]Handler) Dispatcher {
	return &dispatcher{handlers: handlers}
}
