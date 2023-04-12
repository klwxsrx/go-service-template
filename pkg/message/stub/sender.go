package stub

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

type producer struct{}

func (p producer) Send(_ context.Context, _ *message.ProducerMessage) error {
	return nil
}

func (p producer) Close() {}

type producerProvider struct{}

func (p producerProvider) Producer(_ string) (message.Producer, error) {
	return producer{}, nil
}

func NewProducerProvider() message.ProducerProvider {
	return producerProvider{}
}
