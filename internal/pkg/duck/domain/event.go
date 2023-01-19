package domain

import "github.com/google/uuid"

const EventTypeDuckCreated = "duck.duck.created"

type EventDuckCreated struct {
	EventID uuid.UUID `json:"event_id"`
	DuckID  uuid.UUID `json:"duck_id"`
}

func (e EventDuckCreated) ID() uuid.UUID {
	return e.EventID
}

func (e EventDuckCreated) Type() string {
	return EventTypeDuckCreated
}
