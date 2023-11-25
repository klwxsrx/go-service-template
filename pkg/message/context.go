package message

import (
	"context"
)

type contextKey int

const (
	handlerMetaContextKey contextKey = iota
)

type panicErr struct {
	Message    string
	Stacktrace []byte
}

type handlerMetadata struct {
	Panic *panicErr
	Data  Metadata
}

func withHandlerMetadata(ctx context.Context, data Metadata) context.Context {
	if len(data) == 0 {
		data = make(Metadata)
	}
	return context.WithValue(ctx, handlerMetaContextKey, &handlerMetadata{
		Data: data,
	})
}

func getHandlerMetadata(ctx context.Context) *handlerMetadata {
	meta, ok := ctx.Value(handlerMetaContextKey).(*handlerMetadata)
	if ok {
		return meta
	}
	return &handlerMetadata{}
}
