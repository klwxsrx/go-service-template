package http

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
	"net/http"
	"strings"
	"unicode"
)

type (
	ClientOption func(*resty.Client)
)

type Client interface {
	NewRequest(ctx context.Context) *resty.Request
}

type client struct {
	impl *resty.Client
}

func (c client) NewRequest(ctx context.Context) *resty.Request {
	return c.impl.NewRequest().SetContext(ctx)
}

func NewClient(opts ...ClientOption) Client {
	impl := resty.New()
	for _, opt := range opts {
		opt(impl)
	}
	return client{impl}
}

func WithBaseURL(url string) ClientOption {
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
			req.SetHeader(requestIDHeaderName, id)
			return nil
		})
	}
}

func WithRequestLogging(destinationName string, logger log.Logger, infoLevel, errorLevel log.Level) ClientOption {
	return func(r *resty.Client) {
		logger = logger.WithField("destinationName", destinationName)
		r.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
			loggerWithFields := getRequestResponseFieldsLogger(resp.Request.RawRequest, resp.StatusCode(), logger)

			if resp.StatusCode() >= http.StatusInternalServerError {
				loggerWithFields.Log(resp.Request.Context(), errorLevel, "http call completed with internal error")
			} else {
				loggerWithFields.Log(resp.Request.Context(), infoLevel, "http call completed")
			}
			return nil
		})
		r.OnError(func(req *resty.Request, err error) {
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
			key := fmt.Sprintf(
				"client.http.%s.%s.%d",
				prepareDestinationNameForMetrics(destinationName),
				getRouteName(resp.Request.Method, resp.Request.RawRequest.URL.Path),
				resp.StatusCode(),
			)
			metrics.Duration(key, resp.Time())
			return nil
		})
	}
}

func prepareDestinationNameForMetrics(destinationName string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, strings.ToLower(destinationName))
}
