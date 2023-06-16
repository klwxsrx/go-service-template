package http

import (
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"net/http"
	"time"
)

func WithMetrics(metrics metric.Metrics) ServerOption {
	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lrw := &loggingResponseWriter{w, http.StatusOK}

			started := time.Now()
			handler.ServeHTTP(lrw, r)

			metrics.With(metric.Labels{
				"method": r.Method,
				"path":   r.URL.Path,
				"code":   fmt.Sprintf("%d", lrw.code),
			}).Duration("http_api_request_duration_seconds", time.Since(started))
		})
	})
}
