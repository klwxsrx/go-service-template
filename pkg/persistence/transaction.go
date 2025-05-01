package persistence

import "context"

const SkipAlreadyLockedData = LockUpdateOption("skip_already_locked_data")

type (
	Transaction interface {
		WithinContext(ctx context.Context, fn func(context.Context) error, locks ...Lock) error
		LockUpdate(ctx context.Context, exclusively bool, opts ...LockUpdateOption) context.Context
	}

	Lock struct {
		Key    string
		Shared bool
	}

	LockUpdateOption string
)

func WithinTransactionWithResult[T any](
	ctx context.Context,
	tx Transaction,
	fn func(context.Context) (T, error),
	locks ...Lock,
) (T, error) {
	var result T
	err := tx.WithinContext(ctx, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		return err
	}, locks...)

	return result, err
}
