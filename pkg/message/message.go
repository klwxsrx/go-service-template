package message

import (
	"context"

	"github.com/google/uuid"
)

type (
	Message struct {
		ID    uuid.UUID
		Topic Topic
		// Key is used for topic partitioning, messages with the same key will fall in the same topic partition
		Key     string
		Payload []byte
	}

	StructuredMessage interface {
		ID() uuid.UUID
		Type() string
	}

	TypedHandler[T StructuredMessage] func(context.Context, T) error

	KeyBuilder func(StructuredMessage) string

	RegisterMessageFunc func() (
		StructuredMessage,
		KeyBuilder,
	)

	RegisterHandlersFunc func() (
		StructuredMessage,
		PayloadDeserializer,
		[]TypedHandler[StructuredMessage],
	)
)
