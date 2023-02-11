package http

import (
	"context"
	"github.com/google/uuid"
	"net/http"
)

const (
	DefaultRequestIDHeader = "X-Request-ID"
)

type RequestIDExtractor func(r *http.Request) (string, bool)

func WithRequestID(extractor RequestIDExtractor, fallbacks ...RequestIDExtractor) Option {
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

			r = r.WithContext(context.WithValue(r.Context(), requestIDContextKey, requestID))
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

func getIDFromRequest(r *http.Request) (string, bool) {
	requestID, ok := r.Context().Value(requestIDContextKey).(string)
	if !ok || len(requestID) == 0 {
		return "", false
	}
	return requestID, true
}
