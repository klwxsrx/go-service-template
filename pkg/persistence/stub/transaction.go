package stub

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type transaction struct{}

func (s transaction) Execute(ctx context.Context, fn func(ctx context.Context) error, _ ...string) error {
	return fn(ctx)
}

func NewTransaction() persistence.Transaction {
	return &transaction{}
}
