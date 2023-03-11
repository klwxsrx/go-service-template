package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/external"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

const GooseDomainEventTopicName = "goose-domain-event"

type gooseEventDeserializer struct{}

func (g *gooseEventDeserializer) ParseType(msg *message.Message) (string, error) {
	var base baseMessagePayload
	err := json.Unmarshal(msg.Payload, &base)
	if err != nil || base.EventType == "" {
		return "", message.ErrEventDeserializeNotValidEvent
	}
	return base.EventType, nil
}

func (g *gooseEventDeserializer) Deserialize(msg *message.Message) (event.Event, error) {
	eventType, err := g.ParseType(msg)
	if err != nil {
		return nil, err
	}

	switch eventType {
	case external.EventTypeGooseQuacked:
		return g.deserializeGooseQuacked(msg)
	default:
		return nil, fmt.Errorf("unknown event type %s", eventType)
	}
}

func (g *gooseEventDeserializer) deserializeGooseQuacked(msg *message.Message) (event.Event, error) {
	var payload gooseQuackedMessagePayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil || payload.EventID == uuid.Nil || payload.GooseID == uuid.Nil {
		return external.EventGooseQuacked{}, errors.New("invalid message data")
	}

	return external.EventGooseQuacked{
		EventID: payload.EventID,
		GooseID: payload.GooseID,
	}, nil
}

func NewGooseEventDeserializer() message.EventDeserializer {
	return &gooseEventDeserializer{}
}

type gooseQuackedMessagePayload struct {
	baseMessagePayload
	GooseID uuid.UUID `json:"duck_id"`
}

type baseMessagePayload struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
}
