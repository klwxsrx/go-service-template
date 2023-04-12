package http

import (
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"net/http"
)

type PanicHandlerOption func(r *http.Request, panicMsg any)

func NewDefaultPanicHandler(options ...PanicHandlerOption) PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, panicMsg any) {
		w.WriteHeader(http.StatusInternalServerError)

		for _, opt := range options {
			opt(r, panicMsg)
		}
	}
}

func WithPanicLogging(logger log.Logger) PanicHandlerOption { // TODO: stacktrace
	return func(r *http.Request, panicMsg any) {
		getRequestFieldsLogger(r, logger).
			WithField("panic", panicMsg).
			Error(r.Context(), "request handled with panic")
	}
}

func WithPanicMetrics(metrics metric.Metrics) PanicHandlerOption {
	return func(r *http.Request, panicMsg any) {
		metrics.Increment(fmt.Sprintf("app.panic.api.http.%s", getRouteName(r.Method, r.URL.Path)))
	}
}

func panicHandlerWrapper(handler http.HandlerFunc, panicHandler PanicHandler) http.HandlerFunc {
	recoverPanic := func(w http.ResponseWriter, r *http.Request) {
		msg := recover()
		if msg == nil {
			return
		}
		panicHandler(w, r, msg)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		defer recoverPanic(w, r)
		handler.ServeHTTP(w, r)
	}
}
