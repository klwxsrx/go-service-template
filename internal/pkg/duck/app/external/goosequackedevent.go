package external

import "github.com/google/uuid"

const EventTypeGooseQuacked = "goose.quacked"

type EventGooseQuacked struct {
	EventID uuid.UUID
	GooseID uuid.UUID
}

func (e EventGooseQuacked) ID() uuid.UUID {
	return e.EventID
}

func (e EventGooseQuacked) Type() string {
	return EventTypeGooseQuacked
}
