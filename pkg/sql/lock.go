package sql

import (
	"context"
	"fmt"
	"hash/fnv"
)

type lock struct {
	ctx    context.Context
	name   string
	client Client
}

func (l *lock) Get() error {
	return lockDatabase(l.ctx, l.client, "SELECT pg_advisory_lock($1)", l.name)
}

func (l *lock) Release() {
	lockID, _ := getLockIDByName(l.name)
	_, _ = l.client.ExecContext(l.ctx, "SELECT pg_advisory_unlock($1)", lockID)
}

func newLock(ctx context.Context, name string, client Client) *lock {
	return &lock{ctx, name, client}
}

func getLockIDByName(name string) (int64, error) {
	hash := fnv.New64a()
	_, err := hash.Write([]byte(name))
	if err != nil {
		return 0, fmt.Errorf("failed to create name hash for lock: %w", err)
	}
	return int64(hash.Sum64()), nil
}

func lockDatabase(ctx context.Context, client Client, query, lockName string) error {
	lockID, err := getLockIDByName(lockName)
	if err != nil {
		return err
	}

	_, err = client.ExecContext(ctx, query, lockID)
	if err != nil {
		return fmt.Errorf("failed to get lock \"%s\": %w", lockName, err)
	}
	return nil
}
