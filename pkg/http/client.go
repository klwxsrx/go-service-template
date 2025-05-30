package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/klwxsrx/go-service-template/pkg/auth"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

type (
	Destination string

	Route struct {
		Method string
		// URL should be passed as a common URL with path param placeholders, for example /users/{id}/status
		URL string
	}

	Request interface {
		SetPathParam(param, value string) Request
		SetPathParams(params map[string]string) Request
		SetQueryParam(param, value string) Request
		SetQueryParams(params map[string]string) Request
		SetQueryString(query string) Request
		SetHeader(header, value string) Request
		SetHeaders(headers map[string]string) Request
		SetHeaderMultiValues(headers map[string][]string) Request
		SetCookie(cookie *http.Cookie) Request
		SetCookies(cookies []http.Cookie) Request
		SetJSONBody(body any) Request
		SetFormData(data map[string]string) Request
		SetMultipartFormData(data map[string]string) Request
		SetMultipartField(param, fileName, contentType string, reader io.Reader) Request
		Send() (Response, error)
	}

	Response interface {
		StatusCode() int
		Header() http.Header
		Cookies() []http.Cookie
		Body() io.ReadCloser
		ContentLength() int64
		RawRequest() *http.Request
		RawResponse() *http.Response
		Close()
	}

	ClientOption func(*ClientImpl)

	Client interface {
		NewRequest(context.Context, Route) Request
		With(...ClientOption) Client
	}

	ClientImpl struct {
		DestinationName string
		ResponseMWs     []func(*resty.Response, error) error
		RESTClient      *resty.Client
		opts            []ClientOption
	}

	ObservabilityFieldHeaders map[observability.Field]string
)

func NewClient(opts ...ClientOption) Client {
	client := ClientImpl{
		DestinationName: "",
		ResponseMWs:     nil,
		RESTClient:      resty.New(),
		opts:            opts,
	}

	for _, opt := range opts {
		opt(&client)
	}

	return client
}

func (c ClientImpl) NewRequest(ctx context.Context, route Route) Request {
	ctx = context.WithValue(ctx, clientRouteName, getRouteName(route.Method, route.URL))

	r := c.RESTClient.NewRequest().
		SetContext(ctx).
		SetDoNotParseResponse(true)
	r.Method = route.Method
	r.URL = route.URL

	return restyRequestWrapper{
		Request:     r,
		responseMWs: c.ResponseMWs,
	}
}

func (c ClientImpl) With(opts ...ClientOption) Client {
	if len(opts) == 0 {
		return c
	}

	mergedOpts := make([]ClientOption, 0, len(c.opts)+len(opts))
	mergedOpts = append(mergedOpts, c.opts...)
	mergedOpts = append(mergedOpts, opts...)

	return NewClient(mergedOpts...)
}

func WithClientDestination(name, url string) ClientOption {
	return func(c *ClientImpl) {
		c.DestinationName = name
		c.RESTClient.SetBaseURL(url)
	}
}

func WithRequestObservability(
	observer observability.Observer,
	headers ObservabilityFieldHeaders,
) ClientOption {
	return func(c *ClientImpl) {
		c.RESTClient.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			for field, header := range headers {
				value := observer.Field(req.Context(), field)
				if value != "" {
					req.SetHeader(header, value)
				}
			}

			return nil
		})
	}
}

func WithRequestLogging(logger log.Logger, infoLevel, errorLevel log.Level) ClientOption {
	const httpCallLogMsg = "http call"
	return func(c *ClientImpl) {
		c.RESTClient.OnInvalid(func(req *resty.Request, err error) {
			authentication, _ := auth.GetAuthentication[auth.Principal](req.Context())
			logger := getAuthFieldsLogger(authentication, logger)

			routeName, _ := req.Context().Value(clientRouteName).(string)
			if req.RawRequest != nil {
				logger = getResponseFieldsLogger(getDestinationNameForLogging(c), routeName, req.RawRequest, 0, logger)
			} else {
				logger = getRouteNameFieldsLogger(routeName, logger)
			}

			logger.
				WithError(err).
				Log(req.Context(), errorLevel, "invalid request")
		})

		c.RESTClient.OnSuccess(func(_ *resty.Client, resp *resty.Response) {
			authentication, _ := auth.GetAuthentication[auth.Principal](resp.Request.Context())
			logger := getAuthFieldsLogger(authentication, logger)

			routeName, _ := resp.Request.Context().Value(clientRouteName).(string)
			logger = getResponseFieldsLogger(getDestinationNameForLogging(c), routeName, resp.Request.RawRequest, resp.StatusCode(), logger)

			if resp.StatusCode() >= http.StatusInternalServerError {
				logger.Log(resp.Request.Context(), errorLevel, httpCallLogMsg)
			} else {
				logger.Log(resp.Request.Context(), infoLevel, httpCallLogMsg)
			}
		})

		c.RESTClient.OnError(func(req *resty.Request, err error) {
			authentication, _ := auth.GetAuthentication[auth.Principal](req.Context())
			logger := getAuthFieldsLogger(authentication, logger)

			routeName, _ := req.Context().Value(clientRouteName).(string)

			var respError *resty.ResponseError
			switch {
			case errors.As(err, &respError):
				resp := respError.Response
				logger = getResponseFieldsLogger(getDestinationNameForLogging(c), routeName, resp.Request.RawRequest, resp.StatusCode(), logger)
			case req.RawRequest != nil:
				logger = getResponseFieldsLogger(getDestinationNameForLogging(c), routeName, req.RawRequest, 0, logger)
			default:
				logger = getRouteNameFieldsLogger(routeName, logger)
			}

			logger.
				WithError(err).
				Log(req.Context(), errorLevel, httpCallLogMsg)
		})
	}
}

func WithRequestMetrics(metrics metric.Metrics) ClientOption {
	return func(c *ClientImpl) {
		c.RESTClient.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
			var authType *string
			if resp.Request != nil {
				authentication, ok := auth.GetAuthentication[auth.Principal](resp.Request.Context())
				if ok && authentication.Principal() != nil {
					v := string((*authentication.Principal()).Type())
					authType = &v
				}
			}

			destinationName := c.DestinationName
			if destinationName == "" {
				destinationName = "none"
			}

			routeName, _ := resp.Request.Context().Value(clientRouteName).(string)
			if routeName == "" {
				routeName = "none"
			}

			metrics.With(metric.Labels{
				"destination": destinationName,
				"route":       routeName,
				"auth":        authType,
				"method":      resp.Request.Method,
				"path":        resp.Request.RawRequest.URL.Path,
				"code":        fmt.Sprintf("%d", resp.StatusCode()),
			}).Duration("http_client_request_duration_seconds", resp.Time())
			return nil
		})
	}
}

func WithRequestDataFromAuth[T auth.Principal](fn func(auth.Authentication[T], Request)) ClientOption {
	return func(c *ClientImpl) {
		c.RESTClient.OnBeforeRequest(func(_ *resty.Client, r *resty.Request) error {
			authentication, ok := auth.GetAuthentication[T](r.Context())
			if !ok || !authentication.IsAuthenticated() {
				return nil
			}

			fn(authentication, restyRequestWrapper{Request: r})
			return nil
		})
	}
}

func WithRequestHeader(header, value string) ClientOption {
	return func(c *ClientImpl) {
		c.RESTClient.SetHeader(header, value)
	}
}

func WithAuthResponseCodeMapping() ClientOption {
	return WithResponseCodeMapping(map[int]error{
		http.StatusUnauthorized: auth.ErrUnauthenticated,
		http.StatusForbidden:    auth.ErrPermissionDenied,
	})
}

func WithResponseCodeMapping(statusCodes map[int]error) ClientOption {
	if len(statusCodes) == 0 {
		return func(*ClientImpl) {}
	}

	mw := func(resp *resty.Response, err error) error {
		if err != nil {
			return nil
		}

		var ok bool
		if err, ok = statusCodes[resp.StatusCode()]; ok {
			err = fmt.Errorf("%w: response status code %d", err, resp.StatusCode())
		}

		return err
	}

	return func(c *ClientImpl) { c.ResponseMWs = append(c.ResponseMWs, mw) }
}

type ClientFactory struct {
	baseOpts []ClientOption
}

func NewClientFactory(opts ...ClientOption) ClientFactory {
	return ClientFactory{
		baseOpts: opts,
	}
}

func (f *ClientFactory) InitClient(dest Destination, baseURL string, extraOpts ...ClientOption) Client {
	opts := make([]ClientOption, 0, len(extraOpts)+1)
	opts = append(opts, WithClientDestination(string(dest), baseURL))
	opts = append(opts, extraOpts...)

	return f.httpClient(opts...)
}

func (f *ClientFactory) InitRawClient(extraOpts ...ClientOption) Client {
	return f.httpClient(extraOpts...)
}

func (f *ClientFactory) httpClient(extraOpts ...ClientOption) Client {
	opts := make([]ClientOption, 0, len(f.baseOpts)+len(extraOpts))
	opts = append(opts, f.baseOpts...)
	opts = append(opts, extraOpts...)

	return NewClient(opts...)
}

func getDestinationNameForLogging(c *ClientImpl) string {
	if c.DestinationName != "" {
		return c.DestinationName
	}

	return "-"
}

type (
	restyRequestWrapper struct {
		*resty.Request
		responseMWs []func(*resty.Response, error) error
	}

	restyResponseWrapper struct {
		*resty.Response
	}
)

func (r restyRequestWrapper) SetPathParam(param, value string) Request {
	r.Request.SetPathParam(param, value)
	return r
}

func (r restyRequestWrapper) SetPathParams(params map[string]string) Request {
	r.Request.SetPathParams(params)
	return r
}

func (r restyRequestWrapper) SetQueryParam(param, value string) Request {
	r.Request.SetQueryParam(param, value)
	return r
}

func (r restyRequestWrapper) SetQueryParams(params map[string]string) Request {
	r.Request.SetQueryParams(params)
	return r
}

func (r restyRequestWrapper) SetQueryString(query string) Request {
	r.Request.SetQueryString(query)
	return r
}

func (r restyRequestWrapper) SetHeader(header, value string) Request {
	r.Request.SetHeader(header, value)
	return r
}

func (r restyRequestWrapper) SetHeaders(headers map[string]string) Request {
	r.Request.SetHeaders(headers)
	return r
}

func (r restyRequestWrapper) SetHeaderMultiValues(headers map[string][]string) Request {
	r.Request.SetHeaderMultiValues(headers)
	return r
}

func (r restyRequestWrapper) SetCookie(cookie *http.Cookie) Request {
	r.Request.SetCookie(cookie)
	return r
}

func (r restyRequestWrapper) SetCookies(cookies []http.Cookie) Request {
	for _, cookie := range cookies {
		r.Request.SetCookie(&cookie)
	}

	return r
}

func (r restyRequestWrapper) SetJSONBody(body any) Request {
	r.Request.SetHeader("Content-Type", "application/json").SetBody(body)
	return r
}

func (r restyRequestWrapper) SetFormData(data map[string]string) Request {
	r.Request.SetFormData(data)
	return r
}

func (r restyRequestWrapper) SetMultipartFormData(data map[string]string) Request {
	r.Request.SetMultipartFormData(data)
	return r
}

func (r restyRequestWrapper) SetMultipartField(param, fileName, contentType string, reader io.Reader) Request {
	r.Request.SetMultipartField(param, fileName, contentType, reader)
	return r
}

func (r restyRequestWrapper) Send() (Response, error) {
	resp, err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	for _, mw := range r.responseMWs {
		err = mw(resp, err)
	}

	return restyResponseWrapper{resp}, err
}

func (r restyResponseWrapper) Cookies() []http.Cookie {
	restyCookies := r.Response.Cookies()
	result := make([]http.Cookie, 0, len(restyCookies))
	for _, cookie := range restyCookies {
		result = append(result, *cookie)
	}

	return result
}

func (r restyResponseWrapper) Body() io.ReadCloser {
	if r.Response.RawResponse == nil {
		return http.NoBody
	}

	return r.Response.RawResponse.Body
}

func (r restyResponseWrapper) ContentLength() int64 {
	if r.Response.RawResponse == nil {
		return 0
	}

	return r.Response.RawResponse.ContentLength
}

func (r restyResponseWrapper) RawRequest() *http.Request {
	if r.Response.Request != nil {
		return r.Response.Request.RawRequest
	}

	return nil
}

func (r restyResponseWrapper) RawResponse() *http.Response {
	return r.Response.RawResponse
}

func (r restyResponseWrapper) Close() {
	if r.Response.RawResponse != nil {
		_ = r.Response.RawResponse.Body.Close()
	}
}
