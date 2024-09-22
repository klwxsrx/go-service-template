package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"

	"github.com/klwxsrx/go-service-template/pkg/auth"
	"github.com/klwxsrx/go-service-template/pkg/strings"
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

	RequestDataProvider[T any] func(*http.Request) (T, error)

	supportedParsingTypes interface {
		strings.SupportedValueParsingTypes | strings.SupportedPointerParsingTypes
	}
)

var ErrParsingError = errors.New("invalid request data")

func Parse[T any](from *http.Request, data RequestDataProvider[T], lastErr error) (T, error) {
	if lastErr != nil {
		var result T
		return result, lastErr
	}

	return data(from)
}

func ParseOptional[T any](from *http.Request, data RequestDataProvider[T], lastErr error) *T {
	if lastErr != nil {
		return nil
	}

	result, err := data(from)
	if err != nil {
		return nil
	}

	return &result
}

func PathParameter[T supportedParsingTypes](param string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		params := mux.Vars(r)
		paramValue, ok := params[param]
		if !ok {
			var result T
			return result, fmt.Errorf("%w: path parameter %s not found", ErrParsingError, param)
		}
		return parseTypedValueImpl[T](paramValue)
	}
}

func QueryParameter[T supportedParsingTypes](param string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		value := r.URL.Query().Get(param)
		if value == "" {
			var result T
			return result, fmt.Errorf("%w: query parameter %s not found", ErrParsingError, param)
		}
		return parseTypedValueImpl[T](value)
	}
}

func QueryParameters[T supportedParsingTypes](param string) RequestDataProvider[[]T] {
	return func(r *http.Request) ([]T, error) {
		values, ok := r.URL.Query()[param]
		if !ok {
			return nil, fmt.Errorf("%w: query parameter %s not found", ErrParsingError, param)
		}
		result := make([]T, 0, len(values))
		for _, value := range values {
			concreteValue, err := parseTypedValueImpl[T](value)
			if err != nil {
				return nil, err
			}
			result = append(result, concreteValue)
		}
		return result, nil
	}
}

func Header[T supportedParsingTypes](key string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		header := r.Header.Get(key)
		if header == "" {
			var result T
			return result, fmt.Errorf("%w: header with key %s not found", ErrParsingError, key)
		}
		return parseTypedValueImpl[T](header)
	}
}

func Cookie(name string) RequestDataProvider[*http.Cookie] {
	return func(r *http.Request) (*http.Cookie, error) {
		cookie, err := r.Cookie(name)
		if err != nil {
			return nil, fmt.Errorf("%w: cookie with name %s not found", ErrParsingError, name)
		}
		return cookie, nil
	}
}

func CookieValue[T supportedParsingTypes](name string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		cookie, err := r.Cookie(name)
		if err != nil {
			var result T
			return result, fmt.Errorf("%w: cookie with name %s not found", ErrParsingError, name)
		}
		return parseTypedValueImpl[T](cookie.Value)
	}
}

func JSONBody[T any]() RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		var body T
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			return body, fmt.Errorf("%w: encode json body: %w", ErrParsingError, err)
		}
		return body, nil
	}
}

func parseTypedValueImpl[T supportedParsingTypes](value string) (T, error) {
	v, err := strings.ParseTypedValue[T](value)
	if err == nil {
		return v, nil
	}
	return v, fmt.Errorf("%w: %w", ErrParsingError, err)
}

type (
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

func (w *responseWriter) WritePanic(ctx context.Context, panic panicErr) {
	meta := getHandlerMetadata(ctx)
	meta.Code = http.StatusInternalServerError
	meta.Panic = &panic

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
