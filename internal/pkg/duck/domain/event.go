package domain

import "github.com/google/uuid"

const EventTypeDuckCreated = "duck.created"

type EventDuckCreated struct {
	EventID uuid.UUID
	DuckID  uuid.UUID
}

func (e EventDuckCreated) ID() uuid.UUID {
	return e.EventID
}

func (e EventDuckCreated) Type() string {
	return EventTypeDuckCreated
}
