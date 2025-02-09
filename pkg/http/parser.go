package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/klwxsrx/go-service-template/pkg/strings"
)

type (
	DataExtractor[T any] func(dataProvider) (T, error)

	supportedParsingTypes interface {
		strings.SupportedValueParsingTypes | strings.SupportedPointerParsingTypes
	}

	dataProvider interface {
		PathParameters() map[string]string
		QueryParameters() url.Values
		Header() http.Header
		Cookies() []*http.Cookie
		Body() io.ReadCloser
	}

	requestDataProvider struct {
		*http.Request
	}

	responseDataProvider struct {
		*http.Response
	}
)

var ErrParsingError = errors.New("parsing error")

func ParseRequest[T any](r *http.Request, extractor DataExtractor[T], lastErr error) (T, error) {
	if lastErr != nil {
		var result T
		return result, lastErr
	}

	return extractor(requestDataProvider{r})
}

func ParseRequestOptional[T any](r *http.Request, extractor DataExtractor[T], lastErr error) *T {
	if lastErr != nil {
		return nil
	}

	result, err := extractor(requestDataProvider{r})
	if err != nil {
		return nil
	}

	return &result
}

func ParseResponse[T any](r Response, extractor DataExtractor[T], lastErr error) (T, error) {
	if lastErr != nil {
		var result T
		return result, lastErr
	}

	return extractor(responseDataProvider{r.RawResponse()}) //nolint:bodyclose
}

func ParseResponseOptional[T any](r Response, extractor DataExtractor[T], lastErr error) *T {
	if lastErr != nil {
		return nil
	}

	result, err := extractor(responseDataProvider{r.RawResponse()}) //nolint:bodyclose
	if err != nil {
		return nil
	}

	return &result
}

func PathParameter[T supportedParsingTypes](param string) DataExtractor[T] {
	return func(p dataProvider) (T, error) {
		paramValue, ok := p.PathParameters()[param]
		if !ok {
			var result T
			return result, fmt.Errorf("%w: path parameter %s not found", ErrParsingError, param)
		}

		return parseTypedValueImpl[T](paramValue)
	}
}

func QueryParameter[T supportedParsingTypes](param string) DataExtractor[T] {
	return func(p dataProvider) (T, error) {
		value := p.QueryParameters().Get(param)
		if value == "" {
			var result T
			return result, fmt.Errorf("%w: query parameter %s not found", ErrParsingError, param)
		}

		return parseTypedValueImpl[T](value)
	}
}

func QueryParameters[T supportedParsingTypes](param string) DataExtractor[[]T] {
	return func(p dataProvider) ([]T, error) {
		values, ok := p.QueryParameters()[param]
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

func Header[T supportedParsingTypes](key string) DataExtractor[T] {
	return func(p dataProvider) (T, error) {
		header := p.Header().Get(key)
		if header == "" {
			var result T
			return result, fmt.Errorf("%w: header with key %s not found", ErrParsingError, key)
		}

		return parseTypedValueImpl[T](header)
	}
}

func Cookie(name string) DataExtractor[*http.Cookie] {
	return func(p dataProvider) (*http.Cookie, error) {
		var cookie *http.Cookie
		for _, c := range p.Cookies() {
			if c.Name == name {
				cookie = c
			}
		}
		if cookie != nil {
			return cookie, nil
		}

		return nil, fmt.Errorf("%w: cookie with name %s not found", ErrParsingError, name)
	}
}

func CookieValue[T supportedParsingTypes](name string) DataExtractor[T] {
	return func(p dataProvider) (T, error) {
		var cookie *http.Cookie
		for _, c := range p.Cookies() {
			if c.Name == name {
				cookie = c
			}
		}
		if cookie != nil {
			return parseTypedValueImpl[T](cookie.Value)
		}

		var result T
		return result, fmt.Errorf("%w: cookie with name %s not found", ErrParsingError, name)
	}
}

func JSONBody[T any]() DataExtractor[T] {
	return func(p dataProvider) (T, error) {
		var result T
		err := json.NewDecoder(p.Body()).Decode(&result)
		if err != nil {
			return result, fmt.Errorf("%w: encode json body: %w", ErrParsingError, err)
		}

		return result, nil
	}
}

func (p requestDataProvider) PathParameters() map[string]string {
	return mux.Vars(p.Request)
}

func (p requestDataProvider) QueryParameters() url.Values {
	return p.Request.URL.Query()
}

func (p requestDataProvider) Header() http.Header {
	return p.Request.Header
}

func (p requestDataProvider) Body() io.ReadCloser {
	return p.Request.Body
}

func (p responseDataProvider) PathParameters() map[string]string {
	if p.Response == nil || p.Response.Request == nil {
		return nil
	}

	return mux.Vars(p.Response.Request)
}

func (p responseDataProvider) QueryParameters() url.Values {
	if p.Response == nil || p.Response.Request == nil {
		return nil
	}

	return p.Response.Request.URL.Query()
}

func (p responseDataProvider) Header() http.Header {
	if p.Response == nil {
		return nil
	}

	return p.Response.Header
}

func (p responseDataProvider) Body() io.ReadCloser {
	if p.Response == nil {
		return http.NoBody
	}

	return p.Response.Body
}

func parseTypedValueImpl[T supportedParsingTypes](value string) (T, error) {
	v, err := strings.ParseTypedValue[T](value)
	if err == nil {
		return v, nil
	}

	return v, fmt.Errorf("%w: %w", ErrParsingError, err)
}
