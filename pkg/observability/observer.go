package observability

import (
	"context"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

const FieldRequestID Field = "requestID"

type (
	Observer interface {
		Field(context.Context, Field) string
		WithField(context.Context, Field, string) context.Context
	}

	ObserverOption func(*observer)

	Field string
)

type observer struct {
	logger        log.Logger
	loggingFields map[Field]struct{}
}

func New(opts ...ObserverOption) Observer {
	o := observer{}
	for _, opt := range opts {
		opt(&o)
	}

	return o
}

func (o observer) Field(ctx context.Context, field Field) string {
	value, _ := ctx.Value(field).(string)
	return value
}

func (o observer) WithField(ctx context.Context, field Field, value string) context.Context {
	if value == "" {
		return ctx
	}

	ctx = context.WithValue(ctx, field, value)
	if _, ok := o.loggingFields[field]; ok {
		ctx = o.logger.WithContext(ctx, log.Fields{
			string(field): value,
		})
	}

	return ctx
}

func WithFieldsLogging(logger log.Logger, fields ...Field) ObserverOption {
	return func(o *observer) {
		o.logger = logger
		o.loggingFields = make(map[Field]struct{}, len(fields))
		for _, field := range fields {
			o.loggingFields[field] = struct{}{}
		}
	}
}
