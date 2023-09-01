package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/observability"
)

const (
	DefaultRequestIDHeader = "X-Request-ID"
)

type RequestIDExtractor func(r *http.Request) (string, bool)

func WithObservability(
	observer observability.Observer,
	extractor RequestIDExtractor, fallbacks ...RequestIDExtractor,
) ServerOption {
	extractors := append([]RequestIDExtractor{extractor}, fallbacks...)
	findRequestID := func(r *http.Request) (string, bool) {
		for _, ext := range extractors {
			requestID, ok := ext(r)
			if ok {
				return requestID, true
			}
		}
		return "", false
	}

	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID, ok := findRequestID(r)
			if !ok {
				handler.ServeHTTP(w, r)
				return
			}

			ctx := observer.WithRequestID(r.Context(), requestID)
			ctx = withRequestID(ctx, requestID)
			r = r.WithContext(ctx)
			handler.ServeHTTP(w, r)
		})
	})
}

func NewHTTPHeaderRequestIDExtractor(header string) RequestIDExtractor {
	return func(r *http.Request) (string, bool) {
		value := r.Header.Get(header)
		if len(value) > 0 {
			return value, true
		}
		return "", false
	}
}

func NewRandomUUIDRequestIDExtractor() RequestIDExtractor {
	return func(_ *http.Request) (string, bool) {
		return uuid.New().String(), true
	}
}
