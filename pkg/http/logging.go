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

func WithLogging(logger log.Logger, infoLevel, errorLevel log.Level, excludedPaths ...string) ServerOption {
	excludedPaths = append(excludedPaths,
		healthPath,
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
			handler.ServeHTTP(lrw, r)

			loggerWithFields := getRequestResponseFieldsLogger(r, lrw.code, logger)
			if lrw.code >= http.StatusInternalServerError {
				loggerWithFields.Log(r.Context(), errorLevel, "request handled with internal error")
			} else {
				loggerWithFields.Log(r.Context(), infoLevel, "request handled")
			}
		})
	})
}

func getRequestFieldsLogger(r *http.Request, logger log.Logger) log.Logger {
	return logger.With(wrapFieldsWithRequestLogEntry(
		log.Fields{
			"routeName": getRouteName(r.Method, r.URL.Path),
			"method":    r.Method,
			"scheme":    r.URL.Scheme,
			"host":      r.URL.Host,
			"path":      r.URL.Path,
			"rawQuery":  r.URL.RawQuery,
		},
	))
}

func getRequestResponseFieldsLogger(r *http.Request, responseCode int, logger log.Logger) log.Logger {
	return getRequestFieldsLogger(r, logger).With(wrapFieldsWithRequestLogEntry(
		log.Fields{
			"responseCode": responseCode,
		},
	))
}

func wrapFieldsWithRequestLogEntry(fields log.Fields) log.Fields {
	return log.Fields{
		"httpRequest": fields,
	}
}
