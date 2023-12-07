//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Transaction=Transaction"
package persistence

import "context"

type Transaction interface {
	WithinContext(ctx context.Context, fn func(ctx context.Context) error, namedLocks ...string) error
	WithLock(ctx context.Context) context.Context
}
