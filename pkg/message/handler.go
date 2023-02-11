package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
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

type handlerProcess struct { // TODO: WithLogging/WithMetrics option (do with middleware)
	handler  Handler
	consumer Consumer
	logger   log.Logger
}

func NewHandlerProcess(handler Handler, consumer Consumer, logger log.Logger) hub.Process {
	return &handlerProcess{handler, consumer, logger}
}

func (p *handlerProcess) Name() string {
	return fmt.Sprintf("message handler %s", p.consumer.Name())
}

func (p *handlerProcess) Func() func(stopChan <-chan struct{}) error {
	processMessage := func(msg *ConsumerMessage) {
		ctx := p.logger.WithContext(msg.Context, log.Fields{
			"consumerCorrelationID": uuid.New(),
			"consumerMessageID":     msg.Message.ID,
			"consumerTopic":         msg.Message.Topic,
		})

		err := p.handler.Handle(ctx, &msg.Message)
		if err != nil {
			p.consumer.Nack(msg)
			p.logger.WithError(err).Warn(ctx, "failed to handle message")
			return
		}

		p.consumer.Ack(msg)
		p.logger.Info(ctx, "message handled")
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
