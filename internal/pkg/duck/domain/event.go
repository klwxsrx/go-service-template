package domain

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	aggregateNameDuck = "duck"
)

type EventDuckCreated struct {
	EventID uuid.UUID `json:"event_id"`
	DuckID  uuid.UUID `json:"duck_id"`
}

func (e EventDuckCreated) ID() uuid.UUID {
	return e.EventID
}

func (e EventDuckCreated) Type() string {
	return fmt.Sprintf("%s.created", aggregateNameDuck)
}

func (e EventDuckCreated) AggregateID() uuid.UUID {
	return e.DuckID
}

func (e EventDuckCreated) AggregateName() string {
	return aggregateNameDuck
}
