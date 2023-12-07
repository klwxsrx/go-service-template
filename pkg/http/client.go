package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

type (
	Destination string

	ClientOption func(*ClientImpl)

	Client interface {
		NewRequest(ctx context.Context) *resty.Request
		With(opts ...ClientOption) Client
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

func (c ClientImpl) NewRequest(ctx context.Context) *resty.Request {
	return c.RESTClient.NewRequest().SetContext(ctx)
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
			logger = getRequestResponseFieldsLogger(resp.Request.RawRequest, resp.StatusCode(), logger)
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
				logger = getRequestFieldsLogger(req.RawRequest, logger)
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
