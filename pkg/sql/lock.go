package sql

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
)

type lock struct {
	ctx      context.Context
	client   Client
	nameHash uint64
}

func (l *lock) Get() error {
	var success int
	err := l.client.GetContext(l.ctx, &success, "SELECT pg_advisory_lock(?)", l.nameHash)
	if err != nil {
		return fmt.Errorf("failed to get lock: %w", err)
	}
	if success == 0 {
		return errors.New("failed to get lock, attempt timed out")
	}
	return nil
}

func (l *lock) Release() {
	_, _ = l.client.ExecContext(l.ctx, "SELECT pg_advisory_unlock(?)", l.nameHash)
}

func newLock(ctx context.Context, client Client, name string) (*lock, error) {
	hash := fnv.New64a()
	_, err := hash.Write([]byte(name))
	if err != nil {
		return nil, fmt.Errorf("failed to create name hash for lock: %w", err)
	}

	return &lock{ctx, client, hash.Sum64()}, nil
}
