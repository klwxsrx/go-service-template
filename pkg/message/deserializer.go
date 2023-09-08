package message

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrDeserializeUnknownMessage  = errors.New("unknown message type")
	ErrDeserializeNotValidMessage = errors.New("message has not valid struct")
)

type (
	Deserializer interface {
		Deserialize(ctx context.Context, publisherDomain, messageClass string, msg *Message) (StructuredMessage, error)
		RegisterDeserializer(publisherDomain, messageClass, messageType string, deserializer DeserializerFunc) error
	}

	DeserializerFunc func(serializedPayload string) (StructuredMessage, error)
)

type jsonDeserializer struct {
	deserializers map[messageIdentity]DeserializerFunc
}

func newJSONDeserializer() Deserializer {
	return jsonDeserializer{
		deserializers: make(map[messageIdentity]DeserializerFunc),
	}
}

func (d jsonDeserializer) RegisterDeserializer(publisherDomain, messageClass, messageType string, deserializer DeserializerFunc) error {
	id := messageIdentity{
		DomainName:   publisherDomain,
		MessageClass: messageClass,
		MessageType:  messageType,
	}
	if _, ok := d.deserializers[id]; ok {
		return fmt.Errorf("deserializer for %v already exists", id)
	}

	d.deserializers[id] = deserializer
	return nil
}

func (d jsonDeserializer) Deserialize(_ context.Context, publisherDomain, messageClass string, msg *Message) (StructuredMessage, error) {
	var messagePayload jsonPayload
	err := json.Unmarshal(msg.Payload, &messagePayload)
	if err != nil {
		return nil, ErrDeserializeNotValidMessage
	}

	deserializer, ok := d.deserializers[messageIdentity{
		DomainName:   publisherDomain,
		MessageClass: messageClass,
		MessageType:  messagePayload.Type,
	}]
	if !ok {
		return nil, fmt.Errorf("%w %s for domain %s", ErrDeserializeUnknownMessage, messagePayload.Type, publisherDomain)
	}

	return deserializer(messagePayload.Data)
}

func TypedDeserializer[T StructuredMessage]() DeserializerFunc {
	return func(serializedPayload string) (StructuredMessage, error) {
		var result T
		err := json.Unmarshal([]byte(serializedPayload), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize message %T: %w", result, err)
		}
		return result, nil
	}
}
