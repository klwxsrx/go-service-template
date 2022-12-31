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
	Messages() <-chan *ConsumerMessage
	Ack(msg *ConsumerMessage)
	Nack(msg *ConsumerMessage)
	Close()
}

type Handler interface {
	Handle(ctx context.Context, msg *Message) error
}

func NewHandlerProcess(handler Handler, consumer Consumer, optionalLogger log.Logger) hub.Process {
	if optionalLogger == nil {
		optionalLogger = log.NewStub()
	}

	processMessage := func(msg *ConsumerMessage) {
		logger := optionalLogger.With(log.Fields{
			"messageID": msg.Message.ID,
			"topic":     msg.Message.Topic,
		})

		err := handler.Handle(msg.Context, &msg.Message)
		if err != nil {
			consumer.Nack(msg)
			logger.WithError(err).Error(msg.Context, "failed to handle message")
			return
		}

		consumer.Ack(msg)
		logger.Info(msg.Context, "message handled")
	}

	return func(stopChan <-chan struct{}) {
		for {
			select {
			case msg, ok := <-consumer.Messages():
				if !ok {
					return
				}
				processMessage(msg)
			case <-stopChan:
			}
		}
	}
}
