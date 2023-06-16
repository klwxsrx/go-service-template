package message

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"runtime/debug"
)

type Panic struct {
	Message    any
	Stacktrace []byte
}

type (
	PanicHandler       func(context.Context, *Message, Panic) error
	PanicHandlerOption func(context.Context, *Message, Panic)
)

func NewDefaultPanicHandler(options ...PanicHandlerOption) PanicHandler {
	return func(ctx context.Context, message *Message, p Panic) error {
		for _, opt := range options {
			opt(ctx, message, p)
		}
		return fmt.Errorf("message handled with panic: %v", p.Message)
	}
}

func WithPanicLogging(logger log.Logger) PanicHandlerOption {
	return func(ctx context.Context, message *Message, p Panic) {
		logger.WithField("panic", log.Fields{
			"message": p.Message,
			"stack":   string(p.Stacktrace),
		}).Error(ctx, "message handled with panic")
	}
}

func WithPanicMetrics(metrics metric.Metrics) PanicHandlerOption {
	return func(ctx context.Context, message *Message, p Panic) {
		metrics.WithLabel("topic", message.Topic).Increment("msg_handle_panics_total")
	}
}

func panicHandlerWrapper(handler Handler, panicHandler PanicHandler) Handler {
	return func(ctx context.Context, msg *Message) (err error) {
		recoverPanic := func(ctx context.Context, msg *Message) {
			panicMsg := recover()
			if panicMsg == nil {
				return
			}

			p := Panic{
				Message:    panicMsg,
				Stacktrace: debug.Stack(),
			}
			err = panicHandler(ctx, msg, p)
			if err != nil {
				err = fmt.Errorf("message handled with panic: %v", panicMsg)
			}
		}

		defer recoverPanic(ctx, msg)
		return handler(ctx, msg)
	}
}
