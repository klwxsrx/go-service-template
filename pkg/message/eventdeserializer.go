package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

var (
	ErrEventDeserializeNotValidEvent = errors.New("message is not valid event")
)

type EventDeserializer interface {
	ParseType(msg *Message) (string, error)
	Deserialize(msg *Message) (event.Event, error)
}

type (
	RegisterJSONEventFunc func(*jsonEventDeserializer)
	eventDataDeserializer func(string) (event.Event, error)
)

type jsonEventDeserializer struct {
	deserializers map[string]eventDataDeserializer
}

func (d *jsonEventDeserializer) ParseType(msg *Message) (string, error) {
	var messagePayload eventMessagePayload
	err := json.Unmarshal(msg.Payload, &messagePayload)
	if err != nil {
		return "", ErrEventDeserializeNotValidEvent
	}
	return messagePayload.EventType, nil
}

func (d *jsonEventDeserializer) Deserialize(msg *Message) (event.Event, error) {
	var messagePayload eventMessagePayload
	err := json.Unmarshal(msg.Payload, &messagePayload)
	if err != nil {
		return nil, ErrEventDeserializeNotValidEvent
	}

	deserializer, ok := d.deserializers[messagePayload.EventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type %s", messagePayload.EventType)
	}

	return deserializer(messagePayload.EventData)
}

func NewJSONEventDeserializer(jsonEvents ...RegisterJSONEventFunc) EventDeserializer {
	d := &jsonEventDeserializer{deserializers: make(map[string]eventDataDeserializer, len(jsonEvents))}
	for _, jsonEvent := range jsonEvents {
		jsonEvent(d)
	}
	return d
}

func RegisterJSONEvent[T event.Event](eventType string) RegisterJSONEventFunc {
	return func(d *jsonEventDeserializer) {
		d.deserializers[eventType] = func(data string) (event.Event, error) {
			var result T
			err := json.Unmarshal([]byte(data), &result)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize event %s: %w", eventType, err)
			}
			return result, nil
		}
	}
}
