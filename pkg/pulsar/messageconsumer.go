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
	subscriberName message.SubscriberName,
	consumptionType message.ConsumptionType,
) (message.Consumer, error) {
	var typeOption pulsar.SubscriptionType
	switch consumptionType {
	case message.ConsumptionTypeShared:
		typeOption = pulsar.Shared
	case message.ConsumptionTypeExclusive:
		typeOption = pulsar.Failover
	default:
		typeOption = pulsar.Failover
	}

	opts := pulsar.ConsumerOptions{
		Topic:            string(topic),
		SubscriptionName: string(subscriberName),
		Type:             typeOption,
	}

	cons, err := b.client.Subscribe(opts)
	if err != nil {
		return nil, fmt.Errorf("subscribe to topic %s by %s subscriber", topic, subscriberName)
	}

	return newMessageConsumer(cons, topic), nil
}

type messageConsumer struct {
	name string
	impl pulsar.Consumer

	onceDoer *sync.Once
	messages chan *message.ConsumerMessage
}

func newMessageConsumer(pulsarConsumer pulsar.Consumer, subscribedTopic message.Topic) message.Consumer {
	return &messageConsumer{
		name:     fmt.Sprintf("pulsar/%s/%s", pulsarConsumer.Subscription(), subscribedTopic),
		impl:     pulsarConsumer,
		onceDoer: &sync.Once{},
		messages: make(chan *message.ConsumerMessage),
	}
}

func (c *messageConsumer) Name() string {
	return c.name
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

func (c *messageConsumer) Ack(msg *message.ConsumerMessage) {
	messageID, ok := msg.Context.Value(pulsarMessageIDContextKey).(pulsar.MessageID)
	if !ok {
		return
	}

	// single topic pulsar consumer doesn't return any errors
	_ = c.impl.AckID(messageID)
}

func (c *messageConsumer) Nack(msg *message.ConsumerMessage) {
	messageID, ok := msg.Context.Value(pulsarMessageIDContextKey).(pulsar.MessageID)
	if !ok {
		return
	}

	c.impl.NackID(messageID)
}

func (c *messageConsumer) Close() error {
	c.impl.Close()
	return nil
}
