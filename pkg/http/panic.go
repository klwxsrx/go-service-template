package http

import (
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"net/http"
	"runtime/debug"
)

type (
	PanicHandler       func(http.ResponseWriter, *http.Request, Panic)
	PanicHandlerOption func(*http.Request, Panic)
)

type Panic struct {
	Message    any
	Stacktrace []byte
}

func NewDefaultPanicHandler(options ...PanicHandlerOption) PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, p Panic) {
		w.WriteHeader(http.StatusInternalServerError)

		for _, opt := range options {
			opt(r, p)
		}
	}
}

func WithPanicLogging(logger log.Logger) PanicHandlerOption {
	return func(r *http.Request, p Panic) {
		getRequestFieldsLogger(r, logger).
			WithField("panic", map[string]any{
				"message": p.Message,
				"stack":   string(p.Stacktrace),
			}).
			Error(r.Context(), "request handled with panic")
	}
}

func WithPanicMetrics(metrics metric.Metrics) PanicHandlerOption {
	return func(r *http.Request, _ Panic) {
		metrics.Increment(fmt.Sprintf("app.panic.api.http.%s", getRouteName(r.Method, r.URL.Path)))
	}
}

func panicHandlerWrapper(handler http.HandlerFunc, panicHandler PanicHandler) http.HandlerFunc {
	recoverPanic := func(w http.ResponseWriter, r *http.Request) {
		msg := recover()
		if msg == nil {
			return
		}

		p := Panic{
			Message:    msg,
			Stacktrace: debug.Stack(),
		}
		panicHandler(w, r, p)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		defer recoverPanic(w, r)
		handler.ServeHTTP(w, r)
	}
}
