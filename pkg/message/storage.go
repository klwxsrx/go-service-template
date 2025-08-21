package message

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	StorageSpecification struct {
		IDsExcluded       []uuid.UUID
		Topics            []Topic
		ScheduledAtBefore time.Time
		Limit             int
	}

	Storage interface {
		Lock(ctx context.Context, extraKeys ...string) (_ context.Context, release func() error, _ error)
		Find(ctx context.Context, spec *StorageSpecification) ([]Message, error)
		Store(ctx context.Context, scheduledAt time.Time, msgs ...Message) error
		Delete(ctx context.Context, topic Topic, ids ...uuid.UUID) error
	}
)
