package stub

import "github.com/klwxsrx/go-service-template/pkg/worker"

type pool struct{}

func (p pool) Do(j worker.SimpleJob) {
	j()
}

func (p pool) Wait() {}

func NewPool() worker.Pool {
	return pool{}
}
