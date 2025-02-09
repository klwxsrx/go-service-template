package task

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type (
	Task interface {
		ID() uuid.UUID
		Type() string
	}

	Scheduler interface {
		Schedule(ctx context.Context, at time.Time, tasks ...Task) error
	}

	TypedHandler[T Task] func(ctx context.Context, task T) error
	Handler              TypedHandler[Task]
)
