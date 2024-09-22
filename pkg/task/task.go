//go:generate ${TOOLS_BIN}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Task=Task,Scheduler=Scheduler"
package task

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Task interface {
	ID() uuid.UUID
	Type() string
}

type (
	TypedHandler[T Task] func(ctx context.Context, task T) error
	Handler              TypedHandler[Task]
)

type Scheduler interface {
	Schedule(ctx context.Context, at time.Time, tasks ...Task) error
}
