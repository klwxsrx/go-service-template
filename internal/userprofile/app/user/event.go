package user

import (
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
)

type EventUserDeleted struct {
	EventID uuid.UUID     `json:"eventID"`
	UserID  domain.UserID `json:"userID"`
}

func (e EventUserDeleted) ID() uuid.UUID {
	return e.EventID
}

func (e EventUserDeleted) Type() string {
	return "user.deleted"
}

func (e EventUserDeleted) AggregateID() uuid.UUID {
	return e.UserID.UUID
}
