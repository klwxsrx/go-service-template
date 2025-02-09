package persistence

import (
	"context"
)

type transactionStub struct{}

func NewTransactionStub() Transaction {
	return &transactionStub{}
}

func (s transactionStub) WithinContext(ctx context.Context, fn func(ctx context.Context) error, _ ...string) error {
	return fn(ctx)
}

func (s transactionStub) WithLock(ctx context.Context, _ ...LockOption) context.Context {
	return ctx
}
