package message

import "github.com/google/uuid"

const GooseDomainEventTopicName = "goose-domain-event"

type baseMessagePayload struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
}
