package persistence

import "context"

// TODO: fix transactional call to another service with another transactional (use different db transactions)

type Transaction interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error, lockNames ...string) error
}
