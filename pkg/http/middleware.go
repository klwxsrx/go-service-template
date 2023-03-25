package http

import (
	"github.com/gorilla/mux"
)

func WithMW(mw ServerMiddleware) ServerOption {
	return func(srv *server) {
		srv.router.Use(mux.MiddlewareFunc(mw))
	}
}
