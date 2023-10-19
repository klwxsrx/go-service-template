//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Task=Task,Scheduler=Scheduler"
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
	Schedule(ctx context.Context, tasks []Task, at time.Time) error
}
