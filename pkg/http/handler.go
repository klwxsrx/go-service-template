package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	pkgstrings "github.com/klwxsrx/go-service-template/pkg/strings"
)

type ResponseWriter interface {
	SetHeader(key, value string) ResponseWriter
	SetStatusCode(httpCode int) ResponseWriter
	SetCookie(cookie *http.Cookie) ResponseWriter
	SetJSONBody(data any) ResponseWriter
}

type Handler interface {
	Method() string
	Path() string
	HTTPHandler() func(ResponseWriter, *http.Request)
}

type RequestDataProvider[T any] func(*http.Request) (T, error)

func Parse[T any](provider RequestDataProvider[T], from *http.Request, lastErr error) (T, error) {
	if lastErr != nil {
		var result T
		return result, lastErr
	}
	return provider(from)
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
			return fmt.Errorf("failed to encode to json: %w", err)
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

func (w *responseWriter) Write() {
	if w.writeBodyFunc == nil {
		w.impl.WriteHeader(w.httpCode)
		return
	}

	err := w.writeBodyFunc()
	if err != nil {
		w.httpCode = http.StatusInternalServerError
	}
	w.impl.WriteHeader(w.httpCode)
}

func withResponseWriter(handler func(ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respWriter := &responseWriter{
			impl:          w,
			writeBodyFunc: nil,
			httpCode:      http.StatusOK,
		}
		handler(respWriter, r)
		respWriter.Write()
	}
}
