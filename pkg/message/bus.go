package message

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type (
	StructuredMessage interface {
		ID() uuid.UUID
		// Type must be unique string for message class
		Type() string
	}

	RegisterStructuredMessageFunc func(
		domainName string,
	) (messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc, err error)

	Bus interface {
		Produce(ctx context.Context, messageClass string, msg StructuredMessage, scheduleAt time.Time) error
		RegisterMessages(message RegisterStructuredMessageFunc, messages ...RegisterStructuredMessageFunc) error
	}
)

type bus struct {
	domainName string
	storage    OutboxStorage
	serializer jsonSerializer
}

func (b bus) Produce(ctx context.Context, messageClass string, msg StructuredMessage, scheduleAt time.Time) error {
	serializedMsg, err := b.serializer.Serialize(ctx, b.domainName, messageClass, msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message %T: %w", msg, err)
	}

	return b.storage.Store(ctx, []Message{*serializedMsg}, scheduleAt)
}

func (b bus) RegisterMessages(message RegisterStructuredMessageFunc, messages ...RegisterStructuredMessageFunc) error {
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

func NewBus(domainName string, storage OutboxStorage) Bus { // TODO: add observability to pass request id
	return bus{
		domainName: domainName,
		storage:    storage,
		serializer: newJSONSerializer(),
	}
}
