package persistence

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

type messageSender struct {
	store MessageStore
}

func (s *messageSender) Send(ctx context.Context, msg *message.Message) error {
	return s.store.Store(ctx, []message.Message{*msg})
}

func NewMessageSender(store MessageStore) message.Sender {
	return &messageSender{
		store: store,
	}
}
