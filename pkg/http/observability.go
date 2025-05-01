package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/observability"
)

type (
	ObservabilityFieldExtractor  func(*http.Request) string
	ObservabilityFieldExtractors map[observability.Field][]ObservabilityFieldExtractor
)

func WithObservability(
	observer observability.Observer,
	fields ObservabilityFieldExtractors,
) ServerOption {
	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for field, extractors := range fields {
				for _, extractor := range extractors {
					if value := extractor(r); value != "" {
						ctx := observer.WithField(r.Context(), field, value)
						r = r.WithContext(ctx)
						break
					}
				}
			}

			handler.ServeHTTP(w, r)
		})
	})
}

func ObservabilityFieldHeaderExtractor(header string) ObservabilityFieldExtractor {
	return func(r *http.Request) string {
		return r.Header.Get(header)
	}
}

func ObservabilityFieldRandomUUIDExtractor() ObservabilityFieldExtractor {
	return func(_ *http.Request) string {
		return uuid.New().String()
	}
}
