package message

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

const DuckDomainEventTopicName = "duck-domain-event"

type eventSerializer struct{}

func (s *eventSerializer) Serialize(e event.Event) (*message.Message, error) {
	switch e.Type() {
	case domain.EventTypeDuckCreated:
		return s.serializeDuckCreated(e)
	default:
		return nil, fmt.Errorf("%w %v %s", message.ErrEventSerializerUnknownEventType, e.ID(), e.Type())
	}
}

func (s *eventSerializer) Deserialize(msg *message.Message) (event.Event, error) {
	var base baseMessagePayload
	err := json.Unmarshal(msg.Payload, &base)
	if err != nil || base.EventType == "" {
		return nil, fmt.Errorf("%w %v", message.ErrEventSerializerMessageIsNotValidEvent, msg.ID)
	}

	switch base.EventType {
	case domain.EventTypeDuckCreated:
		return s.deserializeDuckCreated(msg)
	default:
		return nil, fmt.Errorf("%w %v %s", message.ErrEventSerializerUnknownEventType, msg.ID, base.EventType)
	}
}

func (s *eventSerializer) serializeDuckCreated(e event.Event) (*message.Message, error) {
	duckCreated, ok := e.(domain.EventDuckCreated)
	if !ok {
		return nil, s.errInvalidEvent(e)
	}

	payload, _ := json.Marshal(duckCreatedMessagePayload{
		baseMessagePayload: baseMessagePayload{
			EventID:   duckCreated.EventID,
			EventType: e.Type(),
		},
		DuckID: duckCreated.DuckID,
	})

	return &message.Message{
		ID:      duckCreated.EventID,
		Topic:   DuckDomainEventTopicName,
		Key:     duckCreated.DuckID.String(),
		Payload: payload,
	}, nil
}

func (s *eventSerializer) deserializeDuckCreated(msg *message.Message) (event.Event, error) {
	var payload duckCreatedMessagePayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil || payload.EventID == uuid.Nil || payload.DuckID == uuid.Nil {
		return nil, s.errInvalidMessageForEvent(msg, domain.EventTypeDuckCreated)
	}

	return domain.EventDuckCreated{
		EventID: payload.EventID,
		DuckID:  payload.DuckID,
	}, nil
}

func (s *eventSerializer) errInvalidEvent(e event.Event) error {
	return fmt.Errorf("invalid event %v, %s expected", e, e.Type())
}

func (s *eventSerializer) errInvalidMessageForEvent(msg *message.Message, expectedEventType string) error {
	return fmt.Errorf("invalid message %v type, %s expected", msg.ID, expectedEventType)
}

func NewEventSerializer() message.EventSerializer {
	return &eventSerializer{}
}

type baseMessagePayload struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
}

type duckCreatedMessagePayload struct {
	baseMessagePayload
	DuckID uuid.UUID `json:"duck_id"`
}
