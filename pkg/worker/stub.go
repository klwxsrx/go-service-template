package worker

import (
	"context"
)

type poolStub struct{}

func NewPoolStub() Pool {
	return poolStub{}
}

func (s poolStub) Do(ctx context.Context, j Job) {
	j(ctx)
}

func (s poolStub) Wait() {}
