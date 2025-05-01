package message

import (
	"context"

	"github.com/google/uuid"
)

const observabilityMetadataKey = "observability"

const handlerMetaContextKey contextKey = iota

type (
	Metadata map[string]any

	handlerMetadata struct {
		MessageID       uuid.UUID
		MessageTopic    Topic
		MessageMetadata Metadata
		Panic           *panicErr
	}

	panicErr struct {
		Message    string
		Stacktrace []byte
	}

	contextKey int
)

func withHandlerMetadata(ctx context.Context, msg *Message, data Metadata) context.Context {
	if len(data) == 0 {
		data = make(Metadata)
	}

	return context.WithValue(ctx, handlerMetaContextKey, &handlerMetadata{
		MessageID:       msg.ID,
		MessageTopic:    msg.Topic,
		MessageMetadata: data,
	})
}

func getHandlerMetadata(ctx context.Context) *handlerMetadata {
	meta, ok := ctx.Value(handlerMetaContextKey).(*handlerMetadata)
	if ok {
		return meta
	}

	return &handlerMetadata{}
}
