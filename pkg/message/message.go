package message

import (
	"context"
	"github.com/google/uuid"
)

type Message struct {
	ID    uuid.UUID
	Topic string
	// Key is used for topic partitioning, messages with the same key will fall in the same topic partition
	Key     string
	Payload []byte
}

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

type Producer interface {
	Send(ctx context.Context, msg *Message) error
}
