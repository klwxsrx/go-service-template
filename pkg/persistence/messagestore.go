package persistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

type MessageStore interface {
	GetBatch(ctx context.Context) ([]message.Message, error)
	Store(ctx context.Context, msgs []message.Message) error
	Delete(ctx context.Context, ids []uuid.UUID) error
}
