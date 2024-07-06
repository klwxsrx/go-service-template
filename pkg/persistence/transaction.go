//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Transaction=Transaction"
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
