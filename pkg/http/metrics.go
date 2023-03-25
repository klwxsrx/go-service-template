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

			key := fmt.Sprintf("api.http.%s.%d", getRouteName(r.Method, r.URL.Path), lrw.code)
			metrics.Duration(key, time.Since(started))
		})
	})
}
