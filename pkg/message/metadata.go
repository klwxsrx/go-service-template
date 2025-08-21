package message

import (
	"context"

	"github.com/google/uuid"
)

const handlerMetaContextKey contextKey = iota

const observabilityMetaKeyPrefix = "observability/"

type (
	Metadata map[string]string

	HandlerMetadata struct {
		MessageID       uuid.UUID
		MessageTopic    Topic
		MessageMetadata Metadata
		Panic           *PanicErr
	}

	PanicErr struct {
		Message    string
		Stacktrace []byte
	}

	contextKey int
)

func withHandlerMetadata(ctx context.Context, msg *Message, data Metadata) context.Context {
	if len(data) == 0 {
		data = make(Metadata)
	}

	return context.WithValue(ctx, handlerMetaContextKey, &HandlerMetadata{
		MessageID:       msg.ID,
		MessageTopic:    msg.Topic,
		MessageMetadata: data,
	})
}

func GetHandlerMetadata(ctx context.Context) *HandlerMetadata {
	meta, ok := ctx.Value(handlerMetaContextKey).(*HandlerMetadata)
	if ok {
		return meta
	}

	return &HandlerMetadata{MessageMetadata: make(Metadata)}
}
