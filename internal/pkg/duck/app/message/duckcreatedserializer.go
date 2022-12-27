package message

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

var EventSerializerDuckCreated = &serializerDuckCreated{}

type serializerDuckCreated struct{}

func (s *serializerDuckCreated) Serialize(e domain.EventDuckCreated) (*message.Message, error) {
	payload, _ := json.Marshal(duckCreatedMessagePayload{
		baseMessagePayload: baseMessagePayload{
			EventID:   e.EventID,
			EventType: e.Type(),
		},
		DuckID: e.DuckID,
	})

	return &message.Message{
		ID:      e.EventID,
		Topic:   DuckDomainEventTopicName,
		Key:     e.DuckID.String(),
		Payload: payload,
	}, nil
}

func (s *serializerDuckCreated) Deserialize(msg *message.Message) (domain.EventDuckCreated, error) {
	var payload duckCreatedMessagePayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil || payload.EventID == uuid.Nil || payload.DuckID == uuid.Nil {
		return domain.EventDuckCreated{}, s.errInvalidMessageForEvent(msg, domain.EventTypeDuckCreated)
	}

	return domain.EventDuckCreated{
		EventID: payload.EventID,
		DuckID:  payload.DuckID,
	}, nil
}

func (s *serializerDuckCreated) errInvalidMessageForEvent(msg *message.Message, expectedEventType string) error {
	return fmt.Errorf("invalid message %v type, %s expected", msg.ID, expectedEventType)
}

type duckCreatedMessagePayload struct {
	baseMessagePayload
	DuckID uuid.UUID `json:"duck_id"`
}
