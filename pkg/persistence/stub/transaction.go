package stub

import (
	"context"

	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type transaction struct{}

func NewTransaction() persistence.Transaction {
	return &transaction{}
}

func (s transaction) WithinContext(ctx context.Context, fn func(ctx context.Context) error, _ ...string) error {
	return fn(ctx)
}

func (s transaction) WithLock(ctx context.Context, _ ...persistence.LockOption) context.Context {
	return ctx
}
