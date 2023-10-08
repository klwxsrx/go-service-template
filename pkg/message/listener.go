package message

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type HandlerMiddleware func(Handler) Handler

type listener struct {
	consumer Consumer
	handler  Handler
}

func NewListener(
	consumer Consumer,
	handler Handler,
	mws ...HandlerMiddleware,
) worker.NamedProcess {
	l := &listener{
		consumer: consumer,
		handler:  handler,
	}

	l.handler = handlerWrapper(l.handler)
	for i := len(mws) - 1; i >= 0; i-- {
		l.handler = mws[i](l.handler)
	}

	return l
}

func (l *listener) Name() string {
	return fmt.Sprintf("message listener %s", l.consumer.Name())
}

func (l *listener) Process() worker.Process {
	processMessage := func(msg *ConsumerMessage) {
		ctx := withHandlerMetadata(msg.Context)
		err := l.handler(ctx, &msg.Message)
		if err != nil {
			l.consumer.Nack(msg)
			return
		}
		l.consumer.Ack(msg)
	}

	return func(stopChan <-chan struct{}) error {
		for {
			select {
			case msg, ok := <-l.consumer.Messages():
				if !ok {
					return errors.New("consumer closed messages channel")
				}
				processMessage(msg)
			case <-stopChan:
				l.consumer.Close()
				return nil
			}
		}
	}
}

func WithLogging(logger log.Logger, infoLevel, errorLevel log.Level) HandlerMiddleware {
	return func(handler Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			ctx = logger.WithContext(ctx, log.Fields{
				"consumerMessage": log.Fields{
					"correlationID": uuid.New(),
					"messageID":     msg.ID,
					"topic":         msg.Topic,
				},
			})

			err := handler(ctx, msg)
			meta := getHandlerMetadata(ctx)
			if meta.Panic != nil {
				logger.WithField("panic", log.Fields{
					"message": meta.Panic.Message,
					"stack":   string(meta.Panic.Stacktrace),
				}).Error(ctx, "message handled with panic")
				return err
			}
			if err != nil {
				logger.WithError(err).Log(ctx, errorLevel, "message handled with error")
				return err
			}

			logger.Log(ctx, infoLevel, "message handled")
			return nil
		}
	}
}

func WithMetrics(metrics metric.Metrics) HandlerMiddleware {
	return func(handler Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			started := time.Now()

			err := handler(ctx, msg)
			meta := getHandlerMetadata(ctx)
			if meta.Panic != nil {
				metrics.WithLabel("topic", msg.Topic).Increment("msg_handle_panics_total")
			}

			metrics.With(metric.Labels{
				"topic":   msg.Topic,
				"success": err == nil,
			}).Duration("msg_handle_duration_seconds", time.Since(started))
			return err
		}
	}
}

func handlerWrapper(handler Handler) Handler {
	return func(ctx context.Context, msg *Message) (err error) {
		recoverPanic := func(ctx context.Context) {
			panicMsg := recover()
			if panicMsg == nil {
				return
			}

			meta := getHandlerMetadata(ctx)
			meta.Panic = &Panic{
				Message:    fmt.Sprintf("%v", panicMsg),
				Stacktrace: debug.Stack(),
			}

			err = fmt.Errorf("message handled with panic: %v", panicMsg)
		}

		defer recoverPanic(ctx)
		return handler(ctx, msg)
	}
}
