package message

// TODO: add mock/stub

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type ProducerMessage struct {
	ID      uuid.UUID
	Key     string
	Payload []byte
}

type Producer interface {
	Send(ctx context.Context, msg *ProducerMessage) error
	Close()
}

type ProducerProvider interface {
	Producer(topic string) (Producer, error)
}

type Sender interface {
	Send(ctx context.Context, msg *Message) error
}

type ProducerSender struct {
	producerProvider ProducerProvider
	topicProducers   map[string]Producer
}

func (s *ProducerSender) Send(ctx context.Context, msg *Message) error {
	producer, err := s.getProducerForTopic(msg.Topic)
	if err != nil {
		return err
	}

	err = producer.Send(ctx, &ProducerMessage{
		ID:      msg.ID,
		Key:     msg.Key,
		Payload: msg.Payload,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (s *ProducerSender) Close() {
	for _, producer := range s.topicProducers {
		producer.Close()
	}
	s.topicProducers = nil
}

func (s *ProducerSender) getProducerForTopic(topic string) (Producer, error) {
	if producer, ok := s.topicProducers[topic]; ok {
		return producer, nil
	}

	producer, err := s.producerProvider.Producer(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get producer for topic %s: %w", topic, err)
	}
	s.topicProducers[topic] = producer
	return producer, nil
}

func NewSender(provider ProducerProvider) *ProducerSender {
	return &ProducerSender{
		producerProvider: provider,
		topicProducers:   make(map[string]Producer),
	}
}
