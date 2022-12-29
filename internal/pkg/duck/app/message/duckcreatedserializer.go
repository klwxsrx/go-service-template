package message

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

func serializeDuckCreated(e domain.EventDuckCreated) (*message.Message, error) {
	payload, err := json.Marshal(duckCreatedMessagePayload{
		baseMessagePayload: baseMessagePayload{
			EventID:   e.EventID,
			EventType: e.Type(),
		},
		DuckID: e.DuckID,
	})
	if err != nil {
		return nil, err
	}

	return &message.Message{
		ID:      e.EventID,
		Topic:   DuckDomainEventTopicName,
		Key:     e.DuckID.String(),
		Payload: payload,
	}, nil
}

func deserializeDuckCreated(msg *message.Message) (domain.EventDuckCreated, error) {
	var payload duckCreatedMessagePayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return domain.EventDuckCreated{}, err
	}
	if payload.EventID == uuid.Nil || payload.DuckID == uuid.Nil {
		return domain.EventDuckCreated{}, errors.New("invalid message data")
	}

	return domain.EventDuckCreated{
		EventID: payload.EventID,
		DuckID:  payload.DuckID,
	}, nil
}

type duckCreatedMessagePayload struct {
	baseMessagePayload
	DuckID uuid.UUID `json:"duck_id"`
}
