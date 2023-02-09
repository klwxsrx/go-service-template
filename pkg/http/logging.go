package http

import (
	"github.com/klwxsrx/go-service-template/pkg/log"
	"net/http"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	code int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func WithLogging(logger log.Logger, excludedPaths ...string) Option {
	excludedPaths = append(excludedPaths,
		HealthPath,
	)

	isExcluded := func(path string) bool {
		for _, excludedPath := range excludedPaths {
			if excludedPath == path {
				return true
			}
		}
		return false
	}

	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isExcluded(r.URL.Path) {
				handler.ServeHTTP(w, r)
				return
			}

			lrw := &loggingResponseWriter{w, http.StatusOK}
			if reqID, ok := getIDFromRequest(r); ok {
				r = r.WithContext(logger.WithContext(r.Context(), log.Fields{
					"httpRequestID": reqID,
				}))
			}
			handler.ServeHTTP(lrw, r)

			logger.With(log.Fields{
				"routeName":    getRouteName(r.Method, r.URL.Path),
				"method":       r.Method,
				"path":         r.URL.Path,
				"uri":          r.RequestURI,
				"responseCode": lrw.code,
			}).Info(r.Context(), "request handled")
		})
	})
}
