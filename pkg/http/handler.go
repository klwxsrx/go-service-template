package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"

	pkgstrings "github.com/klwxsrx/go-service-template/pkg/strings"
)

type HandlerFunc func(w ResponseWriter, r *http.Request) (err error)

type Handler interface {
	Method() string
	Path() string
	HTTPHandler() HandlerFunc
}

type ResponseWriter interface {
	SetHeader(key, value string) ResponseWriter
	SetStatusCode(httpCode int) ResponseWriter
	SetCookie(cookie *http.Cookie) ResponseWriter
	SetJSONBody(data any) ResponseWriter
}

type RequestDataProvider[T any] func(*http.Request) (T, error)

var ErrParsingError = errors.New("failed to parse request")

func Parse[T any](provider RequestDataProvider[T], from *http.Request, lastErr error) (T, error) {
	if lastErr != nil {
		var result T
		return result, lastErr
	}
	result, err := provider(from)
	if err != nil {
		return result, fmt.Errorf("%w: %s", ErrParsingError, err.Error())
	}
	return result, nil
}

func PathParameter[T any](param string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		params := mux.Vars(r)
		paramValue, ok := params[param]
		if !ok {
			var result T
			return result, fmt.Errorf("path parameter %s not found", param)
		}
		return pkgstrings.ParseTypedValue[T](paramValue)
	}
}

func QueryParameter[T any](param string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		value := r.URL.Query().Get(param)
		if value == "" {
			var result T
			return result, fmt.Errorf("query parameter %s not found", param)
		}
		return pkgstrings.ParseTypedValue[T](value)
	}
}

func QueryParameters[T any](param string) RequestDataProvider[[]T] {
	return func(r *http.Request) ([]T, error) {
		values, ok := r.URL.Query()[param]
		if !ok {
			return nil, fmt.Errorf("query parameter %s not found", param)
		}
		result := make([]T, 0, len(values))
		for _, value := range values {
			concreteValue, err := pkgstrings.ParseTypedValue[T](value)
			if err != nil {
				return nil, err
			}
			result = append(result, concreteValue)
		}
		return result, nil
	}
}

func Header[T any](key string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		header := r.Header.Get(key)
		if header == "" {
			var result T
			return result, fmt.Errorf("header with key %s not found", key)
		}
		return pkgstrings.ParseTypedValue[T](header)
	}
}

func Cookie(name string) RequestDataProvider[*http.Cookie] {
	return func(r *http.Request) (*http.Cookie, error) {
		cookie, err := r.Cookie(name)
		if err != nil {
			return nil, fmt.Errorf("cookie with name %s not found", name)
		}
		return cookie, nil
	}
}

func CookieValue[T any](name string) RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		cookie, err := r.Cookie(name)
		if err != nil {
			var result T
			return result, fmt.Errorf("cookie with name %s not found", name)
		}
		return pkgstrings.ParseTypedValue[T](cookie.Value)
	}
}

func JSONBody[T any]() RequestDataProvider[T] {
	return func(r *http.Request) (T, error) {
		var body T
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			return body, fmt.Errorf("failed to encode json body: %w", err)
		}
		return body, nil
	}
}

type responseWriter struct {
	impl http.ResponseWriter

	writeBodyFunc func() error
	httpCode      int
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
	w.writeBodyFunc = func() error {
		bodyEncoded, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to encode body: %w", err)
		}

		_, err = w.impl.Write(bodyEncoded)
		if err != nil {
			return fmt.Errorf("failed to write body: %w", err)
		}

		w.impl.Header().Set("Content-Type", "application/json")
		return nil
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
	case w.writeBodyFunc != nil:
		err = w.writeBodyFunc()
		if err != nil {
			httpCode = http.StatusInternalServerError
			break
		}
		fallthrough
	default:
		httpCode = w.httpCode
	}

	meta := getHandlerMetadata(ctx)
	meta.Code = httpCode
	meta.Error = err

	w.impl.WriteHeader(httpCode)
}

func (w *responseWriter) WritePanic(ctx context.Context, panic Panic) {
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

		respWriter.WritePanic(r.Context(), Panic{
			Message:    fmt.Sprintf("%v", msg),
			Stacktrace: debug.Stack(),
		})
	}

	return func(w http.ResponseWriter, r *http.Request) {
		respWriter := &responseWriter{
			impl:          w,
			writeBodyFunc: nil,
			httpCode:      http.StatusOK,
		}

		defer recoverPanic(r, respWriter)
		err := handler(respWriter, r)
		respWriter.Write(r.Context(), err)
	}
}
