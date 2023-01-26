package message

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"github.com/klwxsrx/go-service-template/pkg/log"
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

func NewHandlerProcess(handler Handler, consumer Consumer, logger log.Logger, logHandledMessages bool) hub.Process {
	processMessage := func(msg *ConsumerMessage) {
		loggerWithFields := logger.With(log.Fields{
			"messageID": msg.Message.ID,
			"topic":     msg.Message.Topic,
		})

		err := handler.Handle(msg.Context, &msg.Message)
		if err != nil {
			consumer.Nack(msg)
			loggerWithFields.WithError(err).Error(msg.Context, "failed to handle message")
			return
		}

		consumer.Ack(msg)
		if logHandledMessages {
			loggerWithFields.Info(msg.Context, "message handled")
		}
	}

	return func(stopChan <-chan struct{}) {
		for {
			select {
			case msg, ok := <-consumer.Messages():
				if !ok {
					logger.WithField("consumerName", consumer.Name()).Error(msg.Context, "consumer closed messages channel")
					return
				}
				processMessage(msg)
			case <-stopChan:
				return
			}
		}
	}
}
