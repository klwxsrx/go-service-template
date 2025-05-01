package persistence

import (
	"context"
)

type transactionStub struct{}

func NewTransactionStub() Transaction {
	return &transactionStub{}
}

func (s transactionStub) WithinContext(ctx context.Context, fn func(ctx context.Context) error, _ ...Lock) error {
	return fn(ctx)
}

func (s transactionStub) LockUpdate(ctx context.Context, _ bool, _ ...LockUpdateOption) context.Context {
	return ctx
}
