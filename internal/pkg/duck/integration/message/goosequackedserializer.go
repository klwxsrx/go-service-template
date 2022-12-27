package message

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/integration"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

var EventSerializerGooseQuacked = &serializerGooseQuacked{}

type serializerGooseQuacked struct{}

func (s *serializerGooseQuacked) Serialize(e integration.EventGooseQuacked) (*message.Message, error) {
	payload, _ := json.Marshal(gooseQuackedMessagePayload{
		baseMessagePayload: baseMessagePayload{
			EventID:   e.EventID,
			EventType: e.Type(),
		},
		GooseID: e.GooseID,
	})

	return &message.Message{
		ID:      e.EventID,
		Topic:   GooseDomainEventTopicName,
		Key:     e.GooseID.String(),
		Payload: payload,
	}, nil
}

func (s *serializerGooseQuacked) Deserialize(msg *message.Message) (integration.EventGooseQuacked, error) {
	var payload gooseQuackedMessagePayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil || payload.EventID == uuid.Nil || payload.GooseID == uuid.Nil {
		return integration.EventGooseQuacked{}, s.errInvalidMessageForEvent(msg, integration.EventTypeGooseQuacked)
	}

	return integration.EventGooseQuacked{
		EventID: payload.EventID,
		GooseID: payload.GooseID,
	}, nil
}

func (s *serializerGooseQuacked) errInvalidMessageForEvent(msg *message.Message, expectedEventType string) error {
	return fmt.Errorf("invalid message %v type, %s expected", msg.ID, expectedEventType)
}

type gooseQuackedMessagePayload struct {
	baseMessagePayload
	GooseID uuid.UUID `json:"duck_id"`
}
