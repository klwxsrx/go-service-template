package message

import "context"

type ConsumptionType string

const (
	ConsumptionTypeSingle ConsumptionType = "single"
	ConsumptionTypeShared ConsumptionType = "shared"
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

type ConsumerProvider interface {
	ProvideConsumer(topic, subscriberName string, consumptionType ConsumptionType) (Consumer, error)
}

type Dispatcher interface {
	Dispatch(ctx context.Context, msg *Message) error
}

type Broker interface {
	ConsumerProvider
	Dispatcher
}
