package sync

import "context"

type CriticalSection interface {
	Execute(ctx context.Context, name string, f func() error) error
}
