package http

import (
	"github.com/gorilla/mux"
)

func WithMW(mw ServerMiddleware) ServerOption {
	return func(router *mux.Router) {
		router.Use(mux.MiddlewareFunc(mw))
	}
}
