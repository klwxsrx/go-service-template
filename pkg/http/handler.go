package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/klwxsrx/go-service-template/pkg/auth"
)

const statusNotSetValue = 0

type (
	HandlerFunc func(ResponseWriter, *http.Request) error

	Handler interface {
		Method() string
		Path() string
		Handle(ResponseWriter, *http.Request) error
	}

	ResponseWriter interface {
		SetHeader(key, value string) ResponseWriter
		SetStatusCode(httpCode int) ResponseWriter
		SetCookie(cookie *http.Cookie) ResponseWriter
		SetJSONBody(data any) ResponseWriter
	}

	responseWriter struct {
		deferredWriter *deferredResponseWriter
		encodeBodyFunc func(http.Header) ([]byte, error)
	}

	deferredResponseWriter struct {
		w        http.ResponseWriter
		r        *http.Request
		httpCode int
		data     []byte
	}
)

func (w *responseWriter) SetHeader(key, value string) ResponseWriter {
	w.deferredWriter.Header().Set(key, value)
	return w
}

func (w *responseWriter) SetStatusCode(httpCode int) ResponseWriter {
	w.deferredWriter.WriteHeader(httpCode)
	return w
}

func (w *responseWriter) SetCookie(cookie *http.Cookie) ResponseWriter {
	http.SetCookie(w.deferredWriter, cookie)
	return w
}

func (w *responseWriter) SetJSONBody(data any) ResponseWriter {
	w.encodeBodyFunc = func(header http.Header) ([]byte, error) {
		bodyEncoded, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}

		header.Set("Content-Type", "application/json")
		return bodyEncoded, nil
	}
	return w
}

func (w *responseWriter) Write(ctx context.Context, err error) {
	switch {
	case w.deferredWriter.IsStatusCodeExplicitlyWritten():
		// do nothing
	case errors.Is(err, ErrParsingError):
		w.deferredWriter.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, auth.ErrUnauthenticated):
		w.deferredWriter.WriteHeader(http.StatusUnauthorized)
	case errors.Is(err, auth.ErrPermissionDenied):
		w.deferredWriter.WriteHeader(http.StatusForbidden)
	case err != nil:
		w.deferredWriter.WriteHeader(http.StatusInternalServerError)
	default:
	}

	var bodyEncoded []byte
	if w.encodeBodyFunc != nil {
		var encodingErr error
		bodyEncoded, encodingErr = w.encodeBodyFunc(w.deferredWriter.Header())
		if encodingErr != nil {
			w.deferredWriter.WriteHeader(http.StatusInternalServerError)
			if err == nil {
				err = encodingErr
			}
		}
	}
	if len(bodyEncoded) > 0 {
		_, _ = w.deferredWriter.Write(bodyEncoded)
	}

	meta := getHandlerMetadata(ctx)
	meta.Code = w.deferredWriter.StatusCode()
	meta.Error = err

	w.deferredWriter.PersistWrite()
}

func (w *responseWriter) WritePanic(ctx context.Context, p panicErr) {
	meta := getHandlerMetadata(ctx)
	meta.Code = http.StatusInternalServerError
	meta.Panic = &p

	w.deferredWriter.WriteHeader(http.StatusInternalServerError)
	w.deferredWriter.ClearData()
	w.deferredWriter.PersistWrite()
}

func newDeferredResponseWriter(w http.ResponseWriter, r *http.Request) *deferredResponseWriter {
	return &deferredResponseWriter{
		w:        w,
		r:        r,
		httpCode: statusNotSetValue,
		data:     nil,
	}
}

func (w *deferredResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *deferredResponseWriter) Write(bytes []byte) (int, error) {
	w.data = bytes
	return len(bytes), nil
}

func (w *deferredResponseWriter) WriteHeader(statusCode int) {
	w.httpCode = statusCode
}

func (w *deferredResponseWriter) StatusCode() int {
	if w.IsStatusCodeExplicitlyWritten() {
		return w.httpCode
	}
	return http.StatusOK
}

func (w *deferredResponseWriter) IsStatusCodeExplicitlyWritten() bool {
	return w.httpCode != statusNotSetValue
}

func (w *deferredResponseWriter) ClearData() {
	w.data = nil
}

func (w *deferredResponseWriter) PersistWrite() {
	if w.IsStatusCodeExplicitlyWritten() {
		w.w.WriteHeader(w.httpCode)
	}

	if len(w.data) == 0 {
		return
	}

	_, err := w.w.Write(w.data)
	if err != nil {
		meta := getHandlerMetadata(w.r.Context())
		if meta.Error == nil {
			meta.Error = fmt.Errorf("write body: %w", err)
		}
	}
}

func httpHandlerWrapper(handler HandlerFunc) http.HandlerFunc {
	recoverPanic := func(r *http.Request, w *responseWriter) {
		msg := recover()
		if msg == nil {
			return
		}

		w.WritePanic(r.Context(), panicErr{
			Message:    fmt.Sprintf("%v", msg),
			Stacktrace: debug.Stack(),
		})
	}

	return func(w http.ResponseWriter, r *http.Request) {
		respWriter := &responseWriter{
			deferredWriter: newDeferredResponseWriter(w, r),
			encodeBodyFunc: nil,
		}

		defer recoverPanic(r, respWriter)
		err := handler(respWriter, r)
		respWriter.Write(r.Context(), err)
	}
}
