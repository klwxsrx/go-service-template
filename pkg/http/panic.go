package http

import (
	"github.com/klwxsrx/go-service-template/pkg/log"
	"net/http"
)

func NewLoggingPanicHandler(logger log.Logger) PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, panicMsg any) {
		getRequestFieldsLogger(r, logger).
			WithField("panic", panicMsg).
			Error(r.Context(), "request handled with panic")
		w.WriteHeader(http.StatusInternalServerError)
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
