package http

import (
	"net/http"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

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

			requestMeta := HandlerMeta(r.Context())
			if requestMeta.RequestID != nil {
				r = r.WithContext(logger.WithContext(r.Context(), log.Fields{"requestID": *requestMeta.RequestID}))
			}

			handler.ServeHTTP(w, r)
			result := HandlerMeta(r.Context())

			loggerWithFields := getRequestResponseFieldsLogger(r, result.ResponseCode, logger)
			switch {
			case result.Panic != nil:
				loggerWithFields.WithField("panic", log.Fields{
					"message": result.Panic.Message,
					"stack":   string(result.Panic.Stacktrace),
				}).Error(r.Context(), "request handled with panic")
			case result.ResponseCode >= http.StatusInternalServerError:
				loggerWithFields.
					WithError(result.Error).
					Log(r.Context(), errorLevel, "request handled with internal error")
			default:
				loggerWithFields.
					WithError(result.Error).
					Log(r.Context(), infoLevel, "request handled")
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
