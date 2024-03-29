package http

import (
	"net/http"

	"github.com/gorilla/mux"

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

			handler.ServeHTTP(w, r)
			result := getHandlerMetadata(r.Context())

			routeName := mux.CurrentRoute(r).GetName()
			loggerWithFields := getRequestResponseFieldsLogger(routeName, r, result.Code, logger)
			switch {
			case result.Panic != nil:
				loggerWithFields.WithField("panic", log.Fields{
					"message": result.Panic.Message,
					"stack":   string(result.Panic.Stacktrace),
				}).Error(r.Context(), "request handled with panic")
			case result.Code >= http.StatusInternalServerError:
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

func getRequestFieldsLogger(routeName string, r *http.Request, logger log.Logger) log.Logger {
	fields := log.Fields{
		"method":   r.Method,
		"scheme":   r.URL.Scheme,
		"host":     r.URL.Host,
		"path":     r.URL.Path,
		"rawQuery": r.URL.RawQuery,
	}
	if routeName != "" {
		fields["routeName"] = routeName
	}

	return logger.With(wrapFieldsWithRequestLogEntry(fields))
}

func getRequestResponseFieldsLogger(routeName string, r *http.Request, responseCode int, logger log.Logger) log.Logger {
	return getRequestFieldsLogger(routeName, r, logger).With(wrapFieldsWithRequestLogEntry(
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
