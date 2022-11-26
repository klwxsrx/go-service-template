package sql

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/sync"
)

type criticalSection struct { // TODO: replace by transactional implementation
	client Client
}

func (s *criticalSection) Execute(ctx context.Context, name string, f func() error) error {
	l, err := newLock(ctx, s.client, name)
	if err != nil {
		return err
	}

	err = l.Get()
	if err != nil {
		return fmt.Errorf("failed to get lock: %w", err)
	}
	defer l.Release()

	return f()
}

func NewCriticalSection(client Client) sync.CriticalSection {
	return &criticalSection{client}
}
