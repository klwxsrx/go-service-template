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

type HandlerMiddleware func(Handler) Handler

type listener struct { // TODO: add panic handler
	handler  Handler
	consumer Consumer
}

func NewListener(handler Handler, consumer Consumer, mws ...HandlerMiddleware) hub.Process {
	hp := &listener{handler, consumer}
	for i := len(mws) - 1; i >= 0; i-- {
		hp.handler = mws[i](hp.handler)
	}
	return hp
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
			if err != nil {
				metrics.Duration(getMetricKey("messaging.handle.%s.failed", msg.Topic), time.Since(started))
				return err
			}

			metrics.Duration(getMetricKey("messaging.handle.%s.success", msg.Topic), time.Since(started))
			return nil
		}
	}
}

func getMetricKey(keyPattern, msgTopic string) string {
	return fmt.Sprintf(
		keyPattern,
		strings.Map(func(r rune) rune {
			if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
				return r
			}
			return '_'
		}, strings.ToLower(msgTopic)),
	)
}
