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

type storeDispatcher struct {
	store Store
}

func (s *storeDispatcher) Dispatch(ctx context.Context, msg *Message) error {
	return s.store.Store(ctx, []Message{*msg})
}

func NewStoreDispatcher(store Store) Dispatcher {
	return &storeDispatcher{
		store: store,
	}
}
