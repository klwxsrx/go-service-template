package http

import (
	"github.com/gorilla/mux"
)

func WithCORSHandler() ServerOption {
	return func(router *mux.Router) {
		router.Use(mux.CORSMethodMiddleware(router))
	}
}
