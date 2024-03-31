package goose

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	AggregateNameGoose = "goose"
)

type EventGooseQuacked struct {
	EventID uuid.UUID `json:"eventID"`
	GooseID uuid.UUID `json:"gooseID"`
}

func (e EventGooseQuacked) ID() uuid.UUID {
	return e.EventID
}

func (e EventGooseQuacked) AggregateID() uuid.UUID {
	return e.GooseID
}

func (e EventGooseQuacked) Type() string {
	return fmt.Sprintf("%s.quacked", AggregateNameGoose)
}
