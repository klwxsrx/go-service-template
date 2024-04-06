package message

import (
	"encoding/json"
	"fmt"
)

type jsonSerializer struct {
	topic       Topic
	serializers map[string]serializerHelper
}

func newJSONSerializer(topic Topic) jsonSerializer {
	return jsonSerializer{
		topic:       topic,
		serializers: make(map[string]serializerHelper),
	}
}

func (s jsonSerializer) RegisterSerializer(messageType string, keyBuilder KeyBuilderFunc) error {
	if _, ok := s.serializers[messageType]; ok {
		return fmt.Errorf("serializer for %v already exists", messageType)
	}

	s.serializers[messageType] = serializerHelper{
		Key: keyBuilder,
	}

	return nil
}

func (s jsonSerializer) Serialize(
	msg StructuredMessage,
	meta Metadata,
) (*Message, error) {
	serializerHelper, ok := s.serializers[msg.Type()]
	if !ok {
		return nil, fmt.Errorf("unknown message type %s", msg.Type())
	}

	keyBuilder := func(_ StructuredMessage) string { return "" }
	if serializerHelper.Key != nil {
		keyBuilder = serializerHelper.Key
	}

	messageEncoded, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("encode message %v %s: %w", msg.ID(), msg.Type(), err)
	}

	payload, err := json.Marshal(jsonPayload{
		Type: msg.Type(),
		Data: string(messageEncoded),
		Meta: meta,
	})
	if err != nil {
		return nil, fmt.Errorf("encode message payload for %s: %w", msg.Type(), err)
	}

	return &Message{
		ID:      msg.ID(),
		Topic:   s.topic,
		Key:     keyBuilder(msg),
		Payload: payload,
	}, nil
}

type (
	serializerHelper struct {
		Key KeyBuilderFunc
	}

	jsonPayload struct {
		Type string   `json:"type"`
		Data string   `json:"data"`
		Meta Metadata `json:"meta,omitempty"`
	}
)
