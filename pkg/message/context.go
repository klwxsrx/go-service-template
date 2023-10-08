package message

import (
	"context"
)

type contextKey int

const (
	handlerMetaContextKey contextKey = iota
)

type Panic struct {
	Message    string
	Stacktrace []byte
}

type handlerMetadata struct {
	Panic *Panic
}

func withHandlerMetadata(ctx context.Context) context.Context {
	return context.WithValue(ctx, handlerMetaContextKey, &handlerMetadata{})
}

func getHandlerMetadata(ctx context.Context) *handlerMetadata {
	meta, ok := ctx.Value(handlerMetaContextKey).(*handlerMetadata)
	if ok {
		return meta
	}
	return &handlerMetadata{}
}
