package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
)

const healthPath = "/healthz"

func WithServerAddress(addr string) ServerOption {
	return func(srv *ServerImpl) {
		srv.Impl.Addr = addr
	}
}

func WithHandlerOptions(opts ...HandlerOption) ServerOption {
	return func(s *ServerImpl) {
		if len(opts) == 0 {
			return
		}

		router := s.Impl.Handler.(*mux.Router)
		for _, opt := range opts {
			opt(router)
		}
	}
}

func WithHealthCheck(customHandlerFunc HandlerFunc) HandlerOption {
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

func WithErrorMapping(statusCodes map[int][]error) HandlerOption {
	statusCodePredicates := make(map[int]func(error) bool, len(statusCodes))
	for statusCode, errs := range statusCodes {
		statusCodePredicates[statusCode] = func(err error) bool {
			for _, expected := range errs {
				if errors.Is(err, expected) {
					return true
				}
			}
			return false
		}
	}

	return WithErrorMappingPredicate(statusCodePredicates)
}

func WithErrorMappingPredicate(statusCodesPredicates map[int]func(error) bool) HandlerOption {
	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(statusCodesPredicates) == 0 {
				handler.ServeHTTP(w, r)
				return
			}

			respWriter := newDeferredResponseWriter(w, r)
			defer respWriter.PersistWrite()

			handler.ServeHTTP(respWriter, r)

			meta := getHandlerMetadata(r.Context())
			if meta.Error == nil || meta.Panic != nil {
				return
			}

			for statusCode, predicate := range statusCodesPredicates {
				if predicate(meta.Error) {
					respWriter.WriteHeader(statusCode)
					meta.Code = statusCode
					return
				}
			}
		})
	})
}

func WithMW(mw HandlerMiddleware) HandlerOption {
	return func(router *mux.Router) {
		router.Use(mux.MiddlewareFunc(mw))
	}
}

func WithCORSHandler() HandlerOption {
	return func(router *mux.Router) {
		router.Use(mux.CORSMethodMiddleware(router))
	}
}
