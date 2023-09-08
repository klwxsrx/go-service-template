package message

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type (
	StructuredMessage interface {
		ID() uuid.UUID
		Type() string
	}

	RegisterStructuredMessageFunc func(
		domainName string,
	) (messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc, err error)

	Bus interface {
		Produce(ctx context.Context, messageClass string, msg StructuredMessage) error
		RegisterMessage(message RegisterStructuredMessageFunc, messages ...RegisterStructuredMessageFunc) error
	}
)

// TODO: Command and Task implementations

type bus struct {
	domainName string
	producer   Producer
	serializer jsonSerializer
}

func (b bus) Produce(ctx context.Context, messageClass string, msg StructuredMessage) error {
	serializedMsg, err := b.serializer.Serialize(ctx, b.domainName, messageClass, msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message %T: %w", msg, err)
	}

	return b.producer.Produce(ctx, serializedMsg)
}

func (b bus) RegisterMessage(message RegisterStructuredMessageFunc, messages ...RegisterStructuredMessageFunc) error {
	messages = append([]RegisterStructuredMessageFunc{message}, messages...)
	for _, registerFunc := range messages {
		messageClass, messageType, topicBuilder, keyBuilder, err := registerFunc(b.domainName)
		if err != nil {
			return fmt.Errorf("failed to register message for domain %s: %w", b.domainName, err)
		}

		err = b.serializer.RegisterSerializer(b.domainName, messageClass, messageType, topicBuilder, keyBuilder)
		if err != nil {
			return fmt.Errorf("failed to register message serializer for message class %s type %s, domain %s: %w", messageClass, messageType, b.domainName, err)
		}
	}
	return nil
}

func NewBus(domainName string, producer Producer) Bus { // TODO: add observability to pass request id
	return bus{
		domainName: domainName,
		producer:   producer,
		serializer: newJSONSerializer(),
	}
}
