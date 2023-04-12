package pulsar

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

func (c *connection) Send(ctx context.Context, msg *message.Message) error {
	producer, err := c.getOrCreateProducer(msg.Topic)
	if err != nil {
		return err
	}

	_, err = producer.Send(ctx, &pulsar.ProducerMessage{
		Payload:    msg.Payload,
		Key:        msg.Key,
		Properties: map[string]string{messageIDPropertyName: msg.ID.String()},
	})

	return err
}

func (c *connection) getOrCreateProducer(topic string) (pulsar.Producer, error) {
	producer, ok := c.producers[topic] // TODO: exclusive access
	if ok {
		return producer, nil
	}

	producer, err := c.client.CreateProducer(pulsar.ProducerOptions{
		Topic: topic,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer for topic %s: %w", topic, err)
	}

	c.producers[topic] = producer
	return producer, nil
}
