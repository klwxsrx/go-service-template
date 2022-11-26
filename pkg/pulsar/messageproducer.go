package pulsar

import (
	"context"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

type messageProducer struct {
	pulsar pulsar.Producer
}

func (p *messageProducer) Send(ctx context.Context, msg *message.ProducerMessage) error {
	_, err := p.pulsar.Send(ctx, &pulsar.ProducerMessage{
		Payload:    msg.Payload,
		Key:        msg.Key,
		Properties: map[string]string{messageIDPropertyName: msg.ID.String()},
	})
	return err
}

func (p *messageProducer) Close() {
	p.pulsar.Close()
}

func newMessageProducer(pulsarProducer pulsar.Producer) message.Producer {
	return &messageProducer{pulsar: pulsarProducer}
}
