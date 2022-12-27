package event

import (
	"context"
	"github.com/google/uuid"
)

type Event interface {
	ID() uuid.UUID
	Type() string
}

type Dispatcher interface {
	Dispatch(ctx context.Context, events []Event) error
}

type Handler[T Event] func(ctx context.Context, event T) error
