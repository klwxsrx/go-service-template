package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"

	"github.com/klwxsrx/go-service-template/pkg/strings"
)

type (
	HandlerFunc func(w ResponseWriter, r *http.Request) (err error)

	Handler interface {
		Method() string
		Path() string
		HTTPHandler() HandlerFunc
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

type responseWriter struct {
	impl http.ResponseWriter

	encodeBodyFunc func(http.Header) ([]byte, error)
	httpCode       int
}

func (w *responseWriter) SetHeader(key, value string) ResponseWriter {
	w.impl.Header().Set(key, value)
	return w
}

func (w *responseWriter) SetStatusCode(httpCode int) ResponseWriter {
	w.httpCode = httpCode
	return w
}

func (w *responseWriter) SetCookie(cookie *http.Cookie) ResponseWriter {
	http.SetCookie(w.impl, cookie)
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
	var httpCode int
	switch {
	case errors.Is(err, ErrParsingError):
		httpCode = http.StatusBadRequest
	case err != nil:
		httpCode = http.StatusInternalServerError
	default:
		httpCode = w.httpCode
	}

	var bodyEncoded []byte
	if w.encodeBodyFunc != nil {
		var encodingErr error
		bodyEncoded, encodingErr = w.encodeBodyFunc(w.impl.Header())
		if encodingErr != nil {
			httpCode = http.StatusInternalServerError
			if err == nil {
				err = encodingErr
			}
		}
	}
	w.impl.WriteHeader(httpCode)

	if len(bodyEncoded) > 0 {
		_, writeBodyErr := w.impl.Write(bodyEncoded)
		if writeBodyErr != nil {
			httpCode = http.StatusInternalServerError
		}
		if err == nil {
			err = writeBodyErr
		}
	}

	meta := getHandlerMetadata(ctx)
	meta.Code = httpCode
	meta.Error = err
}

func (w *responseWriter) WritePanic(ctx context.Context, panic panicErr) {
	meta := getHandlerMetadata(ctx)
	meta.Code = http.StatusInternalServerError
	meta.Panic = &panic

	w.impl.WriteHeader(http.StatusInternalServerError)
}

func httpHandlerWrapper(handler HandlerFunc) http.HandlerFunc {
	recoverPanic := func(r *http.Request, respWriter *responseWriter) {
		msg := recover()
		if msg == nil {
			return
		}

		respWriter.WritePanic(r.Context(), panicErr{
			Message:    fmt.Sprintf("%v", msg),
			Stacktrace: debug.Stack(),
		})
	}

	return func(w http.ResponseWriter, r *http.Request) {
		respWriter := &responseWriter{
			impl:           w,
			encodeBodyFunc: nil,
			httpCode:       http.StatusOK,
		}

		defer recoverPanic(r, respWriter)
		err := handler(respWriter, r)
		respWriter.Write(r.Context(), err)
	}
}
