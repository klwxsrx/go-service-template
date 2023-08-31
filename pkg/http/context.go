package http

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

type contextKey int

const (
	handlerMetaContextKey contextKey = iota
)

type Panic struct {
	Message    string
	Stacktrace []byte
}

type HandlerMetadata struct {
	RequestID    *string
	ResponseCode int
	Panic        *Panic
	Error        error
}

func withHandlerMetadata(router *mux.Router) *mux.Router {
	router.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), handlerMetaContextKey, &HandlerMetadata{})
			handler.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	return router
}

func HandlerMeta(ctx context.Context) *HandlerMetadata {
	meta, ok := ctx.Value(handlerMetaContextKey).(*HandlerMetadata)
	if ok {
		return meta
	}
	return &HandlerMetadata{}
}
