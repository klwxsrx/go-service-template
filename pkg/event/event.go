package event

import (
	"github.com/google/uuid"
)

type Event interface {
	ID() uuid.UUID
	Type() string
}
