package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

var (
	errEventDeserializeUnknownEvent  = errors.New("unknown event type")
	errEventDeserializeNotValidEvent = errors.New("message is not valid event")
)

type (
	registerTypedEventFunc func(domainName string, deserializer *jsonEventDeserializer) error

	eventDataDeserializer func(string) (event.Event, error)
)

type jsonEventDeserializer struct {
	deserializers map[eventDomainData]eventDataDeserializer
}

func (d *jsonEventDeserializer) Deserialize(domainName string, msg *Message) (event.Event, error) {
	var messagePayload eventMessagePayload
	err := json.Unmarshal(msg.Payload, &messagePayload)
	if err != nil {
		return nil, errEventDeserializeNotValidEvent
	}

	deserializer, ok := d.deserializers[eventDomainData{
		DomainName: domainName,
		EventType:  messagePayload.EventType,
	}]
	if !ok {
		return nil, fmt.Errorf("%w %s for domain %s", errEventDeserializeUnknownEvent, messagePayload.EventType, domainName)
	}

	return deserializer(messagePayload.EventData)
}

func (d *jsonEventDeserializer) RegisterJSONEvent(domainName string, fn registerTypedEventFunc) error {
	return fn(domainName, d)
}

func newJSONEventDeserializer() *jsonEventDeserializer {
	return &jsonEventDeserializer{deserializers: make(map[eventDomainData]eventDataDeserializer, 0)}
}

func registerDeserializerTyped[T event.Event]() registerTypedEventFunc {
	return func(domainName string, d *jsonEventDeserializer) error {
		var blankEvent T
		eventType := blankEvent.Type()
		if eventType == "" {
			return fmt.Errorf("failed to get event type for %T: blank event must return const value", blankEvent)
		}

		d.deserializers[eventDomainData{
			DomainName: domainName,
			EventType:  eventType,
		}] = func(data string) (event.Event, error) {
			var result T
			err := json.Unmarshal([]byte(data), &result)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize event %s: %w", eventType, err)
			}
			return result, nil
		}
		return nil
	}
}

type eventDomainData struct {
	DomainName string
	EventType  string
}
