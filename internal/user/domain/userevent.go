package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type EventUserDeleted struct {
	EventID uuid.UUID `json:"eventID"`
	UserID  UserID    `json:"userID"`
}

func (e EventUserDeleted) ID() uuid.UUID {
	return e.EventID
}

func (e EventUserDeleted) Type() string {
	return fmt.Sprintf("%s.deleted", AggregateNameUser)
}

func (e EventUserDeleted) AggregateID() uuid.UUID {
	return e.UserID.UUID
}
