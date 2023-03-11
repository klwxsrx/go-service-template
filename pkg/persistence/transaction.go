//go:generate mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Transaction=Transaction"
package persistence

import "context"

type Transaction interface {
	Execute(ctx context.Context, fn func(ctx context.Context) error, lockNames ...string) error
}
