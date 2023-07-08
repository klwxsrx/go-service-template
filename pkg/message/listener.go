package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/worker"
	"time"
)

type HandlerMiddleware func(Handler) Handler

type listener struct {
	handler  Handler
	consumer Consumer
}

func NewListener(
	handler Handler,
	consumer Consumer,
	panicHandler PanicHandler,
	mws ...HandlerMiddleware,
) worker.NamedProcess {
	l := &listener{handler, consumer}
	l.handler = panicHandlerWrapper(l.handler, panicHandler)
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
		err := l.handler(msg.Context, &msg.Message)
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
			if err != nil {
				logger.WithError(err).Log(ctx, errorLevel, "failed to handle message")
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
			metrics.With(metric.Labels{
				"topic":   msg.Topic,
				"success": err == nil,
			}).Duration("msg_handle_duration_seconds", time.Since(started))
			return err
		}
	}
}
