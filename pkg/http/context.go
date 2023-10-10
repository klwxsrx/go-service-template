package http

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

type contextKey int

const (
	handlerMetaContextKey contextKey = iota
	clientMetaContextKey
)

type Panic struct {
	Message    string
	Stacktrace []byte
}

type handlerMetadata struct {
	RequestID *string
	Code      int
	Panic     *Panic
	Error     error
}

type clientMetadata struct {
	RequestID *string
}

func withHandlerMetadata(router *mux.Router) *mux.Router {
	router.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), handlerMetaContextKey, &handlerMetadata{})
			handler.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	return router
}

func getHandlerMetadata(ctx context.Context) *handlerMetadata {
	meta, ok := ctx.Value(handlerMetaContextKey).(*handlerMetadata)
	if ok {
		return meta
	}
	return &handlerMetadata{}
}

func withClientMetadata(ctx context.Context, meta *clientMetadata) context.Context {
	return context.WithValue(ctx, clientMetaContextKey, meta)
}

func getClientMetadata(ctx context.Context) *clientMetadata {
	meta, ok := ctx.Value(clientMetaContextKey).(*clientMetadata)
	if ok {
		return meta
	}
	return &clientMetadata{}
}
