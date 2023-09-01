package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/klwxsrx/go-service-template/pkg/metric"
)

func WithMetrics(metrics metric.Metrics) ServerOption {
	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			handler.ServeHTTP(w, r)
			result := getHandlerMetadata(r.Context())

			if result.Panic != nil {
				metrics.With(metric.Labels{
					"method": r.Method,
					"path":   r.URL.Path,
				}).Increment("http_api_request_panics_total")
			}

			metrics.With(metric.Labels{
				"method": r.Method,
				"path":   r.URL.Path,
				"code":   fmt.Sprintf("%d", result.Code),
			}).Duration("http_api_request_duration_seconds", time.Since(started))
		})
	})
}
