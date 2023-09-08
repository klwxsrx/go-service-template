package pulsar

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/klwxsrx/go-service-template/pkg/message"
)

func (b *MessageBroker) Produce(ctx context.Context, msg *message.Message) error {
	producer, err := b.getOrCreateProducer(msg.Topic)
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

func (b *MessageBroker) getOrCreateProducer(topic string) (pulsar.Producer, error) {
	topicMutex, _ := b.producerMutexes.LoadOrStore(topic, &sync.Mutex{})
	topicMutex.(*sync.Mutex).Lock()
	defer topicMutex.(*sync.Mutex).Unlock()

	producer, ok := b.producers[topic]
	if ok {
		return producer, nil
	}

	producer, err := b.client.CreateProducer(pulsar.ProducerOptions{
		Topic: topic,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer for topic %s: %w", topic, err)
	}

	b.producers[topic] = producer
	return producer, nil
}
