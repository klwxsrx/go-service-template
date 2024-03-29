package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	healthPath = "/healthz"
)

func WithHealthCheck(customHandlerFunc HandlerFunc) ServerOption {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ContentType", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Status string `json:"status"`
		}{
			Status: "OK",
		})
	}
	if customHandlerFunc != nil {
		handler = httpHandlerWrapper(customHandlerFunc)
	}

	return func(router *mux.Router) {
		router.
			Name(getRouteName(http.MethodGet, healthPath)).
			Methods(http.MethodGet).
			Path(healthPath).
			HandlerFunc(handler)
	}
}

func WithMW(mw ServerMiddleware) ServerOption {
	return func(router *mux.Router) {
		router.Use(mux.MiddlewareFunc(mw))
	}
}

func WithCORSHandler() ServerOption {
	return func(router *mux.Router) {
		router.Use(mux.CORSMethodMiddleware(router))
	}
}
