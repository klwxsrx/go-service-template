package stub

import "github.com/klwxsrx/go-service-template/pkg/worker"

type pool struct{}

func NewPool() worker.Pool {
	return pool{}
}

func (p pool) Do(j worker.Job) {
	j()
}

func (p pool) Wait() {}
