package http

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	healthPath = "/healthz"
)

func WithHealthCheck(customHandlerFunc http.HandlerFunc) Option {
	defaultHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ContentType", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Status string `json:"status"`
		}{
			Status: "OK",
		})
	}

	return func(router *mux.Router) {
		handler := defaultHandler
		if customHandlerFunc != nil {
			handler = customHandlerFunc
		}

		router.
			Name(getRouteName(http.MethodGet, healthPath)).
			Methods(http.MethodGet).
			Path(healthPath).
			HandlerFunc(handler)
	}
}
