//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Observer=Observer"
package observability

import (
	"context"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

type (
	LogField string

	contextKey int
)

const (
	LogFieldRequestID LogField = "requestID"
)

const (
	requestIDContextKey contextKey = iota
)

type (
	Observer interface {
		RequestID(context.Context) (string, bool)
		WithRequestID(context.Context, string) context.Context
	}

	ObserverOption func(*observer)
)

type observer struct {
	logger        log.Logger
	loggingFields map[LogField]struct{}
}

func New(opts ...ObserverOption) Observer {
	o := observer{}
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

func (o observer) RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok || len(requestID) == 0 {
		return "", false
	}

	return requestID, true
}

func (o observer) WithRequestID(ctx context.Context, id string) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey, id)

	if _, ok := o.loggingFields[LogFieldRequestID]; ok {
		ctx = o.logger.WithContext(ctx, log.Fields{
			string(LogFieldRequestID): id,
		})
	}

	return ctx
}

func WithFieldsLogging(logger log.Logger, fields ...LogField) ObserverOption {
	return func(o *observer) {
		o.logger = logger

		o.loggingFields = make(map[LogField]struct{}, len(fields))
		for _, field := range fields {
			o.loggingFields[field] = struct{}{}
		}
	}
}
