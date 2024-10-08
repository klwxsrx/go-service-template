package http

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/klwxsrx/go-service-template/pkg/auth"
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
			logger := getAuthFieldsLogger(result.Auth, logger)
			logger = getResponseFieldsLogger("", routeName, r, result.Code, logger)
			switch {
			case result.Panic != nil:
				logger.WithField("panic", log.Fields{
					"message": result.Panic.Message,
					"stack":   string(result.Panic.Stacktrace),
				}).Error(r.Context(), "request handled with panic")
			case result.Code >= http.StatusInternalServerError:
				logger.
					WithError(result.Error).
					Log(r.Context(), errorLevel, "request handled with internal error")
			default:
				logger.
					WithError(result.Error).
					Log(r.Context(), infoLevel, "request handled")
			}
		})
	})
}

func getAuthFieldsLogger(
	authentication auth.Authentication[auth.Principal],
	logger log.Logger,
) log.Logger {
	if authentication == nil {
		return logger
	}

	var principalType, principalID *string
	if authentication.Principal() != nil {
		principalTypeValue := string((*authentication.Principal()).Type())
		principalType = &principalTypeValue
		principalID = (*authentication.Principal()).ID()
	}

	fields := log.Fields{
		"type": principalType,
		"id":   principalID,
	}

	return logger.WithField("auth", fields)
}

func getResponseFieldsLogger(
	destinationName string,
	routeName string,
	request *http.Request,
	responseCode int,
	logger log.Logger,
) log.Logger {
	fields := log.Fields{
		"method":   request.Method,
		"scheme":   request.URL.Scheme,
		"host":     request.URL.Host,
		"path":     request.URL.Path,
		"rawQuery": request.URL.RawQuery,
	}
	if destinationName != "" {
		fields["destination"] = destinationName
	}
	if routeName != "" {
		fields["route"] = routeName
	}
	if responseCode != 0 {
		fields["responseCode"] = responseCode
	}

	return logger.WithField("httpRequest", fields)
}

func getRouteNameFieldsLogger(routeName string, logger log.Logger) log.Logger {
	if routeName == "" {
		return logger
	}

	return logger.WithField("httpRequest", log.Fields{"route": routeName})
}
