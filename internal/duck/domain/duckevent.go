package domain

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	AggregateNameDuck = "duck"
)

type EventDuckCreated struct {
	EventID uuid.UUID `json:"eventID"`
	DuckID  uuid.UUID `json:"duckID"`
}

func (e EventDuckCreated) ID() uuid.UUID {
	return e.EventID
}

func (e EventDuckCreated) AggregateID() uuid.UUID {
	return e.DuckID
}

func (e EventDuckCreated) Type() string {
	return fmt.Sprintf("%s.created", AggregateNameDuck)
}
