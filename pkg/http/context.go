package http

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/klwxsrx/go-service-template/pkg/auth"
)

type contextKey int

const (
	handlerMetaContextKey contextKey = iota
	clientRouteName
)

type panicErr struct {
	Message    string
	Stacktrace []byte
}

type handlerMetadata struct {
	Auth  auth.Authentication[auth.Principal]
	Code  int
	Panic *panicErr
	Error error
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
