package message

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	errDeserializeUnknownMessage  = errors.New("unknown message type")
	errDeserializeNotValidMessage = errors.New("message has not valid struct")
)

type (
	DeserializerFunc func(payload []byte) (StructuredMessage, error)

	jsonDeserializer struct {
		deserializers map[string]DeserializerFunc
	}
)

func newJSONDeserializer() jsonDeserializer {
	return jsonDeserializer{
		deserializers: make(map[string]DeserializerFunc),
	}
}

func (d jsonDeserializer) Register(messageType string, deserializer DeserializerFunc) error {
	if _, ok := d.deserializers[messageType]; ok {
		return fmt.Errorf("deserializer for %v already exists", messageType)
	}

	d.deserializers[messageType] = deserializer
	return nil
}

func (d jsonDeserializer) Deserialize(payload []byte) (StructuredMessage, Metadata, error) {
	var messagePayload jsonPayload
	err := json.Unmarshal(payload, &messagePayload)
	if err != nil {
		return nil, nil, errDeserializeNotValidMessage
	}

	deserializer, ok := d.deserializers[messagePayload.Type]
	if !ok {
		return nil, nil, fmt.Errorf("%w %s", errDeserializeUnknownMessage, messagePayload.Type)
	}

	message, err := deserializer([]byte(messagePayload.Data))
	if err != nil {
		return nil, nil, err
	}

	return message, messagePayload.Meta, nil
}

func TypedJSONDeserializer[T StructuredMessage]() DeserializerFunc {
	return func(payload []byte) (StructuredMessage, error) {
		var result T
		err := json.Unmarshal(payload, &result)
		if err != nil {
			return nil, fmt.Errorf("deserialize message %T: %w", result, err)
		}

		return result, nil
	}
}
