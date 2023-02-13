package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"strings"
	"time"
	"unicode"
)

type ConsumerMessage struct {
	Context context.Context
	Message Message
}

type Consumer interface {
	Name() string
	Messages() <-chan *ConsumerMessage
	Ack(msg *ConsumerMessage)
	Nack(msg *ConsumerMessage)
	Close()
}

type Handler interface {
	Handle(ctx context.Context, msg *Message) error
}

type HandlerFunc func(context.Context, *Message) error

func (f HandlerFunc) Handle(ctx context.Context, msg *Message) error {
	return f(ctx, msg)
}

type Middleware func(Handler) Handler

type handlerProcess struct {
	handler  Handler
	consumer Consumer
}

func NewHandlerProcess(handler Handler, consumer Consumer, mws ...Middleware) hub.Process {
	hp := &handlerProcess{handler, consumer}
	for i := len(mws) - 1; i >= 0; i-- {
		hp.handler = mws[i](hp.handler)
	}
	return hp
}

func (p *handlerProcess) Name() string {
	return fmt.Sprintf("message handler %s", p.consumer.Name())
}

func (p *handlerProcess) Func() func(stopChan <-chan struct{}) error {
	processMessage := func(msg *ConsumerMessage) {
		err := p.handler.Handle(msg.Context, &msg.Message)
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
				return nil
			}
		}
	}
}

func WithLogging(logger log.Logger) Middleware {
	return func(handler Handler) Handler {
		return HandlerFunc(func(ctx context.Context, msg *Message) error {
			ctx = logger.WithContext(ctx, log.Fields{
				"consumerCorrelationID": uuid.New(),
				"consumerMessageID":     msg.ID,
				"consumerTopic":         msg.Topic,
			})

			err := handler.Handle(ctx, msg)
			if err != nil {
				logger.WithError(err).Warn(ctx, "failed to handle message")
				return err
			}

			logger.Info(ctx, "message handled")
			return nil
		})
	}
}

func WithMetrics(metrics metric.Metrics) Middleware {
	return func(handler Handler) Handler {
		return HandlerFunc(func(ctx context.Context, msg *Message) error {
			started := time.Now()
			err := handler.Handle(ctx, msg)
			if err != nil {
				metrics.Duration(getMetricKey("async.msg.%s.failed", msg.Topic), time.Since(started))
				return err
			}

			metrics.Duration(getMetricKey("async.msg.%s.success", msg.Topic), time.Since(started))
			return nil
		})
	}
}

func getMetricKey(keyPattern, msgTopic string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, strings.ToLower(fmt.Sprintf(keyPattern, msgTopic)))
}
