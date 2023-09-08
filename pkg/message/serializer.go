package message

import (
	"context"
	"encoding/json"
	"fmt"
)

type (
	TopicBuilderFunc func(domainName string) string
	KeyBuilderFunc   func(StructuredMessage) string
)

type jsonSerializer struct {
	serializers map[messageIdentity]serializerHelper
}

func newJSONSerializer() jsonSerializer {
	return jsonSerializer{
		serializers: make(map[messageIdentity]serializerHelper),
	}
}

func (s jsonSerializer) RegisterSerializer(domainName, messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc) error {
	id := messageIdentity{
		DomainName:   domainName,
		MessageClass: messageClass,
		MessageType:  messageType,
	}
	if _, ok := s.serializers[id]; ok {
		return fmt.Errorf("serializer for %v already exists", id)
	}

	s.serializers[id] = serializerHelper{
		Topic: topicBuilder,
		Key:   keyBuilder,
	}
	return nil
}

func (s jsonSerializer) Serialize(_ context.Context, domainName, messageClass string, msg StructuredMessage) (*Message, error) {
	serializer, ok := s.serializers[messageIdentity{
		DomainName:   domainName,
		MessageClass: messageClass,
		MessageType:  msg.Type(),
	}]
	if !ok {
		return nil, fmt.Errorf("unknown message type %s for domain %s", msg.Type(), domainName)
	}

	messageEncoded, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message %v %s: %w", msg.ID(), msg.Type(), err)
	}

	payload, err := json.Marshal(jsonPayload{
		Type: msg.Type(),
		Data: string(messageEncoded),
		Meta: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode message payload for %s: %w", msg.Type(), err)
	}

	return &Message{
		ID:      msg.ID(),
		Topic:   serializer.Topic(domainName),
		Key:     serializer.Key(msg),
		Payload: payload,
	}, nil
}

type (
	messageIdentity struct {
		DomainName   string
		MessageClass string
		MessageType  string
	}

	serializerHelper struct {
		Topic TopicBuilderFunc
		Key   KeyBuilderFunc
	}

	jsonPayload struct {
		Type string         `json:"type"`
		Data string         `json:"data"`
		Meta map[string]any `json:"meta,omitempty"`
	}
)
