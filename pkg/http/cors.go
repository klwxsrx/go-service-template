package http

import (
	"github.com/gorilla/mux"
)

func WithCORSHandler() ServerOption {
	return func(srv *server) {
		srv.router.Use(mux.CORSMethodMiddleware(srv.router))
	}
}
