package http

import (
	"github.com/gorilla/mux"
)

func WithMW(mw Middleware) Option {
	return func(router *mux.Router) {
		router.Use(mux.MiddlewareFunc(mw))
	}
}
