package pulsar

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/message"
)

const pulsarMessageIDContextKey contextKey = iota

type contextKey int

func (b *MessageBroker) Consumer(
	topic message.Topic,
	subscriber message.Subscriber,
) (message.Consumer[message.AckNackStrategy], error) {
	opts := pulsar.ConsumerOptions{
		Topic:            string(topic),
		SubscriptionName: string(subscriber),
	}

	cons, err := b.client.Subscribe(opts)
	if err != nil {
		return nil, fmt.Errorf("subscribe to topic %s by %s subscriber", topic, subscriber)
	}

	return newMessageConsumer(cons, topic), nil
}

type messageConsumer struct {
	topic      message.Topic
	subscriber message.Subscriber
	impl       pulsar.Consumer

	onceDoer *sync.Once
	messages chan *message.ConsumerMessage
}

func newMessageConsumer(pulsarConsumer pulsar.Consumer, subscribedTopic message.Topic) message.Consumer[message.AckNackStrategy] {
	return &messageConsumer{
		topic:      subscribedTopic,
		subscriber: message.Subscriber(pulsarConsumer.Subscription()),
		impl:       pulsarConsumer,
		onceDoer:   &sync.Once{},
		messages:   make(chan *message.ConsumerMessage),
	}
}

func (c *messageConsumer) Topic() message.Topic {
	return c.topic
}

func (c *messageConsumer) Subscriber() message.Subscriber {
	return c.subscriber
}

func (c *messageConsumer) Messages() <-chan *message.ConsumerMessage {
	messageHandler := func() {
		for {
			msg, ok := <-c.impl.Chan()
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
					Topic:   message.Topic(msg.Topic()),
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

func (c *messageConsumer) Acknowledge() message.AckNackStrategy {
	return c
}

func (c *messageConsumer) Ack(_ context.Context, msg *message.ConsumerMessage) error {
	messageID, ok := msg.Context.Value(pulsarMessageIDContextKey).(pulsar.MessageID)
	if !ok {
		return nil
	}

	err := c.impl.AckID(messageID)
	if err != nil {
		return fmt.Errorf("ack pulsar message id %v: %w", messageID, err)
	}

	return nil
}

func (c *messageConsumer) Nack(_ context.Context, msg *message.ConsumerMessage) error {
	messageID, ok := msg.Context.Value(pulsarMessageIDContextKey).(pulsar.MessageID)
	if !ok {
		return nil
	}

	c.impl.NackID(messageID)
	return nil
}

func (c *messageConsumer) Close() error {
	c.impl.Close()
	return nil
}
