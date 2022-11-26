package http

import (
	"github.com/gorilla/mux"
)

func WithCORSHandler() Option {
	return func(router *mux.Router) {
		router.Use(mux.CORSMethodMiddleware(router))
	}
}
