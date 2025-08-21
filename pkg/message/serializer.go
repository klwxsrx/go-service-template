package message

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrDeserializeUnknownMessage = errors.New("unknown message")

type (
	Serializer interface {
		Serialize(StructuredMessage, Metadata) ([]byte, error)
	}

	Deserializer interface {
		Deserialize([]byte) (StructuredMessage, Metadata, error)
		RegisterDeserializer(msgType string, _ PayloadDeserializer) error
	}

	JSONSerializer struct {
		deserializers map[string]PayloadDeserializer
	}

	PayloadDeserializer func([]byte) (StructuredMessage, error)

	jsonMessage struct {
		Type    string   `json:"type"`
		Payload string   `json:"payload"`
		Meta    Metadata `json:"meta,omitempty"`
	}
)

func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		deserializers: make(map[string]PayloadDeserializer),
	}
}

func (s *JSONSerializer) Serialize(msg StructuredMessage, meta Metadata) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode message %v %s: %w", msg.ID(), msg.Type(), err)
	}

	msgData, err := json.Marshal(jsonMessage{
		Type:    msg.Type(),
		Payload: string(payload),
		Meta:    meta,
	})
	if err != nil {
		return nil, fmt.Errorf("encode message data for %s: %w", msg.Type(), err)
	}

	return msgData, nil
}

func (s *JSONSerializer) Deserialize(data []byte) (StructuredMessage, Metadata, error) {
	var msgData jsonMessage
	err := json.Unmarshal(data, &msgData)
	if err != nil {
		return nil, nil, ErrDeserializeUnknownMessage
	}

	deserializer, ok := s.deserializers[msgData.Type]
	if !ok {
		return nil, nil, fmt.Errorf("%w %s", ErrDeserializeUnknownMessage, msgData.Type)
	}

	msg, err := deserializer([]byte(msgData.Payload))
	if err != nil {
		return nil, nil, fmt.Errorf("deserialize message: %w", err)
	}

	return msg, msgData.Meta, nil
}

func (s *JSONSerializer) RegisterDeserializer(msgType string, deserializer PayloadDeserializer) error {
	if _, ok := s.deserializers[msgType]; ok {
		return fmt.Errorf("deserializer for %v already exists", msgType)
	}

	s.deserializers[msgType] = deserializer
	return nil
}

func PayloadDeserializerImpl[T StructuredMessage](payload []byte) (StructuredMessage, error) {
	var msg T
	err := json.Unmarshal(payload, &msg)
	if err != nil {
		return nil, fmt.Errorf("json decode %T: %w", msg, err)
	}

	return msg, nil
}
