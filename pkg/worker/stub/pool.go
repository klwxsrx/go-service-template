package stub

import "github.com/klwxsrx/go-service-template/pkg/worker"

type pool struct{}

func (p pool) Do(j worker.Job) error {
	j()
	return nil
}

func (p pool) Wait() {}

func (p pool) Close() {}

func NewPool() worker.Pool {
	return pool{}
}
