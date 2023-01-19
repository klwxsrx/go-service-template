package message

import (
	"encoding/json"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type EventSerializer interface {
	Serialize(evt event.Event) (*Message, error)
}

type jsonEventSerializer struct {
	topicName string
}

func (s *jsonEventSerializer) Serialize(evt event.Event) (*Message, error) {
	eventEncoded, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("failed to encode event %s: %w", evt.Type(), err)
	}

	messagePayload, err := json.Marshal(eventMessagePayload{
		EventType: evt.Type(),
		EventData: string(eventEncoded),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode message payload for event %s: %w", evt.Type(), err)
	}

	return &Message{
		ID:      evt.ID(),
		Topic:   s.topicName,
		Key:     "",
		Payload: messagePayload,
	}, nil
}

func NewJSONEventSerializer(topicName string) EventSerializer {
	return &jsonEventSerializer{topicName: topicName}
}

type eventMessagePayload struct {
	EventType string `json:"event_type"`
	EventData string `json:"event_data"`
}
