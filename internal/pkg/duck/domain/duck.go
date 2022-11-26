package domain

import (
	"context"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type Duck struct {
	ID      uuid.UUID
	Changes []event.Event
}

type DuckRepo interface {
	Store(ctx context.Context, duck *Duck) error
}

func NewDuck(id uuid.UUID) *Duck {
	return &Duck{
		ID: id,
		Changes: []event.Event{&EventDuckCreated{
			EventID: uuid.New(),
			DuckID:  id,
		}},
	}
}
