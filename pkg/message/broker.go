package message

import "context"

const (
	ConsumptionTypeSingle ConsumptionType = "single"
	ConsumptionTypeShared ConsumptionType = "shared"
)

type (
	ConsumerMessage struct {
		Context context.Context
		Message Message
	}

	Consumer interface {
		Name() string
		Messages() <-chan *ConsumerMessage
		Ack(msg *ConsumerMessage)
		Nack(msg *ConsumerMessage)
		Close()
	}

	ConsumerProvider interface {
		Consumer(Topic, SubscriberName, ConsumptionType) (Consumer, error)
	}

	Producer interface {
		Produce(ctx context.Context, msg *Message) error
	}

	Broker interface {
		ConsumerProvider
		Producer
	}

	ConsumptionType string
)
