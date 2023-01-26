package pulsar

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"sync"
)

type messageConsumer struct {
	name   string
	pulsar pulsar.Consumer

	onceDoer *sync.Once
	messages chan *message.ConsumerMessage
}

func (c *messageConsumer) Name() string {
	return c.name
}

func (c *messageConsumer) Messages() <-chan *message.ConsumerMessage {
	messageHandler := func() {
		for {
			msg, ok := <-c.pulsar.Chan()
			if !ok {
				close(c.messages)
				break
			}

			ctx := context.WithValue(context.Background(), pulsarMessageIDContextKey, msg.ID())
			messageIDStr, ok := msg.Properties()[messageIDPropertyName]
			if !ok {
				continue
			}
			messageID, err := uuid.Parse(messageIDStr)
			if err != nil {
				continue
			}

			c.messages <- &message.ConsumerMessage{
				Context: ctx,
				Message: message.Message{
					ID:      messageID,
					Topic:   msg.Topic(),
					Key:     msg.Key(),
					Payload: msg.Payload(),
				},
			}
		}
	}

	c.onceDoer.Do(func() {
		go messageHandler()
	})
	return c.messages
}

func (c *messageConsumer) Ack(msg *message.ConsumerMessage) {
	messageID, ok := msg.Context.Value(pulsarMessageIDContextKey).(pulsar.MessageID)
	if !ok {
		return
	}

	c.pulsar.AckID(messageID)
}

func (c *messageConsumer) Nack(msg *message.ConsumerMessage) {
	messageID, ok := msg.Context.Value(pulsarMessageIDContextKey).(pulsar.MessageID)
	if !ok {
		return
	}

	c.pulsar.NackID(messageID)
}

func (c *messageConsumer) Close() {
	c.pulsar.Close()
}

func newMessageConsumer(pulsarConsumer pulsar.Consumer, subscribedTopic string) message.Consumer {
	return &messageConsumer{
		name:     fmt.Sprintf("%s/%s", pulsarConsumer.Subscription(), subscribedTopic),
		pulsar:   pulsarConsumer,
		onceDoer: &sync.Once{},
		messages: make(chan *message.ConsumerMessage),
	}
}
