package persistence

import "context"

type Transaction interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error, lockNames ...string) error
}
