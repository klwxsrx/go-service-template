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

type storeProducer struct {
	store Store
}

func (s *storeProducer) Send(ctx context.Context, msg *Message) error {
	return s.store.Store(ctx, []Message{*msg})
}

func NewStoreProducer(store Store) Producer {
	return &storeProducer{
		store: store,
	}
}
