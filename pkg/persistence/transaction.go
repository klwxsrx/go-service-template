package persistence

import "context"

const SkipAlreadyLockedData = LockOption("skip_already_locked_data")

type (
	Transaction interface {
		WithinContext(ctx context.Context, fn func(context.Context) error, lockNames ...string) error
		WithLock(ctx context.Context, opts ...LockOption) context.Context
	}

	LockOption string
)

func WithinTransactionWithResult[T any](
	ctx context.Context,
	tx Transaction,
	fn func(context.Context) (T, error),
	lockNames ...string,
) (T, error) {
	var result T
	err := tx.WithinContext(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}, lockNames...)

	return result, err
}
