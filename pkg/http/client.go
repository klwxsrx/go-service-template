package http

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

type (
	Destination string

	Route struct {
		Method string
		// URL should be passed as a common URL with path param placeholders, for example /duck/{id}/status
		URL string
	}

	Request interface {
		SetPathParam(param, value string) Request
		SetPathParams(params map[string]string) Request
		SetQueryParam(param, value string) Request
		SetQueryParams(params map[string]string) Request
		SetHeader(header, value string) Request
		SetHeaders(headers map[string]string) Request
		SetHeaderMultiValues(headers map[string][]string) Request
		SetCookie(cookie *http.Cookie) Request
		SetCookies(cookies []http.Cookie) Request
		SetJSONBody(body any) Request
		SetFormData(data map[string]string) Request
		SetMultipartField(param, fileName, contentType string, reader io.Reader) Request
		Send() (*http.Response, error)
	}

	ClientOption func(*ClientImpl)

	Client interface {
		NewRequest(context.Context, Route) Request
		With(...ClientOption) Client
	}

	ClientImpl struct {
		DestinationName string
		RESTClient      *resty.Client
		opts            []ClientOption
	}
)

func NewClient(opts ...ClientOption) Client {
	client := ClientImpl{
		DestinationName: "",
		RESTClient:      resty.New(),
		opts:            opts,
	}

	for _, opt := range opts {
		opt(&client)
	}

	return client
}

func (c ClientImpl) NewRequest(ctx context.Context, route Route) Request {
	r := c.RESTClient.NewRequest().SetContext(ctx)
	r.Method = route.Method
	r.URL = route.URL
	return restyRequestWrapper{r}
}

func (c ClientImpl) With(opts ...ClientOption) Client {
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

func WithRequestObservability(observer observability.Observer, requestIDHeaderName string) ClientOption {
	return func(c *ClientImpl) {
		c.RESTClient.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			id, ok := observer.RequestID(req.Context())
			if !ok {
				return nil
			}

			req.SetHeader(requestIDHeaderName, id)
			return nil
		})
	}
}

func WithRequestLogging(logger log.Logger, infoLevel, errorLevel log.Level) ClientOption {
	const destinationNameLogField = "destinationName"
	return func(c *ClientImpl) {
		c.RESTClient.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
			routeName := getRouteName(resp.Request.Method, resp.Request.URL)
			logger = getRequestResponseFieldsLogger(routeName, resp.Request.RawRequest, resp.StatusCode(), logger)
			logger = logger.With(wrapFieldsWithRequestLogEntry(log.Fields{
				destinationNameLogField: getDestinationNameForLogging(c),
			}))

			if resp.StatusCode() >= http.StatusInternalServerError {
				logger.Log(resp.Request.Context(), errorLevel, "http call completed with internal error")
			} else {
				logger.Log(resp.Request.Context(), infoLevel, "http call completed")
			}

			return nil
		})

		c.RESTClient.OnError(func(req *resty.Request, err error) {
			if req.RawRequest != nil {
				routeName := getRouteName(req.Method, req.URL)
				logger = getRequestFieldsLogger(routeName, req.RawRequest, logger)
			}
			logger = logger.With(wrapFieldsWithRequestLogEntry(log.Fields{
				destinationNameLogField: getDestinationNameForLogging(c),
			}))

			logger.
				WithError(err).
				Log(req.Context(), errorLevel, "http call completed with error")
		})
	}
}

func WithRequestMetrics(metrics metric.Metrics) ClientOption {
	return func(c *ClientImpl) {
		destinationName := c.DestinationName
		if destinationName == "" {
			destinationName = "none"
		}

		c.RESTClient.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			metrics.With(metric.Labels{
				"destination": destinationName,
				"method":      resp.Request.Method,
				"path":        resp.Request.RawRequest.URL.Path,
				"code":        fmt.Sprintf("%d", resp.StatusCode()),
			}).Duration("http_client_request_duration_seconds", resp.Time())
			return nil
		})
	}
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

type restyRequestWrapper struct {
	*resty.Request
}

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
		cookie := cookie
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

func (r restyRequestWrapper) SetMultipartField(param, fileName, contentType string, reader io.Reader) Request {
	r.Request.SetMultipartField(param, fileName, contentType, reader)
	return r
}

func (r restyRequestWrapper) Send() (*http.Response, error) {
	resp, err := r.Request.Send()
	return resp.RawResponse, err
}
