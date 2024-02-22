package stub

import (
	"context"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type pool struct{}

func NewPool() worker.Pool {
	return pool{}
}

func (p pool) Do(ctx context.Context, j worker.Job) {
	j(ctx)
}

func (p pool) Wait() {}
