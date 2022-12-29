package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/integration"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

const GooseDomainEventTopicName = "goose-domain-event"

type gooseEventSerializer struct {
}

func (g *gooseEventSerializer) ParseType(msg *message.Message) (string, error) {
	var base baseMessagePayload
	err := json.Unmarshal(msg.Payload, &base)
	if err != nil || base.EventType == "" {
		return "", message.ErrEventDeserializeNotValidEvent
	}
	return base.EventType, nil
}

func (g *gooseEventSerializer) Deserialize(msg *message.Message) (event.Event, error) {
	eventType, err := g.ParseType(msg)
	if err != nil {
		return nil, err
	}

	switch eventType {
	case integration.EventTypeGooseQuacked:
		return g.deserializeGooseQuacked(msg)
	default:
		return nil, fmt.Errorf("%w, %s", message.ErrEventDeserializeUnknownEventType, eventType)
	}
}

func (g *gooseEventSerializer) deserializeGooseQuacked(msg *message.Message) (event.Event, error) {
	var payload gooseQuackedMessagePayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil || payload.EventID == uuid.Nil || payload.GooseID == uuid.Nil {
		return integration.EventGooseQuacked{}, errors.New("invalid message data")
	}

	return integration.EventGooseQuacked{
		EventID: payload.EventID,
		GooseID: payload.GooseID,
	}, nil
}

func NewGooseEventSerializer() message.EventDeserializer {
	return &gooseEventSerializer{}
}

type gooseQuackedMessagePayload struct {
	baseMessagePayload
	GooseID uuid.UUID `json:"duck_id"`
}

type baseMessagePayload struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
}
