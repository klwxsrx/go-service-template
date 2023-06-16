package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
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
) hub.Process {
	l := &listener{handler, consumer}
	l.handler = panicHandlerWrapper(l.handler, panicHandler)
	for i := len(mws) - 1; i >= 0; i-- {
		l.handler = mws[i](l.handler)
	}
	return l
}

func (p *listener) Name() string {
	return fmt.Sprintf("message listener %s", p.consumer.Name())
}

func (p *listener) Func() func(stopChan <-chan struct{}) error {
	processMessage := func(msg *ConsumerMessage) {
		err := p.handler(msg.Context, &msg.Message)
		if err != nil {
			p.consumer.Nack(msg)
			return
		}
		p.consumer.Ack(msg)
	}

	return func(stopChan <-chan struct{}) error {
		for {
			select {
			case msg, ok := <-p.consumer.Messages():
				if !ok {
					return errors.New("consumer closed messages channel")
				}
				processMessage(msg)
			case <-stopChan:
				p.consumer.Close()
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
