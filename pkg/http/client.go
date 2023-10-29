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
	ClientOption func(*resty.Client)
)

type Client interface {
	NewRequest(ctx context.Context) *resty.Request
	With(opts ...ClientOption) Client
}

type client struct {
	impl *resty.Client
	opts []ClientOption
}

func (c client) NewRequest(ctx context.Context) *resty.Request {
	return c.impl.NewRequest().SetContext(ctx)
}

func (c client) With(opts ...ClientOption) Client {
	mergedOpts := make([]ClientOption, 0, len(c.opts)+len(opts))
	mergedOpts = append(mergedOpts, c.opts...)
	mergedOpts = append(mergedOpts, opts...)
	return NewClient(mergedOpts...)
}

func NewClient(opts ...ClientOption) Client {
	impl := resty.New()
	for _, opt := range opts {
		opt(impl)
	}
	return client{impl, opts}
}

func WithClientBaseURL(url string) ClientOption {
	return func(r *resty.Client) {
		r.SetBaseURL(url)
	}
}

func WithRequestObservability(observer observability.Observer, requestIDHeaderName string) ClientOption {
	return func(r *resty.Client) {
		r.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			id, ok := observer.RequestID(req.Context())
			if !ok {
				return nil
			}

			req.SetContext(withClientMetadata(req.Context(), &clientMetadata{
				RequestID: &id,
			}))
			req.SetHeader(requestIDHeaderName, id)
			return nil
		})
	}
}

func WithRequestLogging(destinationName string, logger log.Logger, infoLevel, errorLevel log.Level) ClientOption {
	logger = logger.With(wrapFieldsWithRequestLogEntry(log.Fields{
		"destinationName": destinationName,
	}))

	return func(r *resty.Client) {
		r.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
			logger = getRequestResponseFieldsLogger(resp.Request.RawRequest, resp.StatusCode(), logger)
			logger = getRequestIDFieldLogger(resp.Request.Context(), logger)
			if resp.StatusCode() >= http.StatusInternalServerError {
				logger.Log(resp.Request.Context(), errorLevel, "http call completed with internal error")
			} else {
				logger.Log(resp.Request.Context(), infoLevel, "http call completed")
			}
			return nil
		})
		r.OnError(func(req *resty.Request, err error) {
			logger = getRequestIDFieldLogger(req.Context(), logger)
			if req.RawRequest != nil {
				logger = getRequestFieldsLogger(req.RawRequest, logger)
			}
			logger.
				WithError(err).
				Log(req.Context(), errorLevel, "http call completed with error")
		})
	}
}

func WithRequestMetrics(destinationName string, metrics metric.Metrics) ClientOption {
	return func(r *resty.Client) {
		r.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
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

func getRequestIDFieldLogger(ctx context.Context, logger log.Logger) log.Logger {
	clientMeta := getClientMetadata(ctx)
	if clientMeta.RequestID == nil {
		return logger
	}
	return logger.WithField(requestIDLogEntry, *clientMeta.RequestID)
}
