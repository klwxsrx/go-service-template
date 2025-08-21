package message

import "context"

type (
	ConsumerMessage struct {
		Context context.Context
		Message Message
	}

	Consumer[S AcknowledgeStrategy] interface {
		Topic() Topic
		Subscriber() Subscriber
		Messages() <-chan *ConsumerMessage
		Acknowledge() S
		Close() error
	}

	ConsumerProvider[S AcknowledgeStrategy] interface {
		Consumer(Topic, Subscriber) (Consumer[S], error)
	}

	Producer interface {
		Produce(context.Context, *Message) error
	}

	Broker[S AcknowledgeStrategy] interface {
		ConsumerProvider[S]
		Producer
		Close() error
	}

	CommitOffsetStrategy interface {
		CommitOffset(context.Context, *ConsumerMessage) error
	}

	AckStrategy interface {
		Ack(context.Context, *ConsumerMessage) error
	}

	AckNackStrategy interface {
		AckStrategy
		Nack(context.Context, *ConsumerMessage) error
	}

	AcknowledgeStrategy any
)
