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

func NewEventSerializer() message.EventSerializer {
	return &eventSerializer{}
}

func (s *eventSerializer) ParseType(msg *message.Message) (string, error) {
	var base baseMessagePayload
	err := json.Unmarshal(msg.Payload, &base)
	if err != nil || base.EventType == "" {
		return "", message.ErrEventDeserializeNotValidEvent
	}
	return base.EventType, nil
}

func (s *eventSerializer) Serialize(event event.Event) (*message.Message, error) {
	switch concreteEvent := event.(type) {
	case *domain.EventDuckCreated:
		return serializeDuckCreated(*concreteEvent)
	default:
		return nil, fmt.Errorf("%w, %s", message.ErrEventSerializeUnknownEventType, event.Type())
	}
}

func (s *eventSerializer) Deserialize(msg *message.Message) (event.Event, error) {
	eventType, err := s.ParseType(msg)
	if err != nil {
		return nil, err
	}

	switch eventType {
	case domain.EventTypeDuckCreated:
		return deserializeDuckCreated(msg)
	default:
		return nil, fmt.Errorf("%w, %s", message.ErrEventDeserializeUnknownEventType, eventType)
	}
}

type baseMessagePayload struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
}
