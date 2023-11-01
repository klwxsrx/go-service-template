package message

import (
	"context"
)

type contextKey int

const (
	handlerMetaContextKey contextKey = iota
	busMetaContextKey
)

type Panic struct {
	Message    string
	Stacktrace []byte
}

type handlerMetadata struct {
	Panic *Panic
	Data  Metadata
}

type busMetadata struct {
	RequestID *string
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

func withBusMetadata(ctx context.Context) context.Context {
	return context.WithValue(ctx, busMetaContextKey, &busMetadata{})
}

func getBusMetadata(ctx context.Context) *busMetadata {
	meta, ok := ctx.Value(busMetaContextKey).(*busMetadata)
	if ok {
		return meta
	}
	return &busMetadata{}
}
