package message

import (
	"context"
	"github.com/google/uuid"
)

type Store interface {
	GetBatch(ctx context.Context) ([]Message, error)
	Store(ctx context.Context, msgs []Message) error
	Delete(ctx context.Context, ids []uuid.UUID) error
}

type storeSender struct {
	store Store
}

func (s *storeSender) Send(ctx context.Context, msg *Message) error {
	return s.store.Store(ctx, []Message{*msg})
}

func NewStoreSender(store Store) Sender {
	return &storeSender{
		store: store,
	}
}
