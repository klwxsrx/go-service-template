package sql

import (
	"context"
	"fmt"
	"hash/fnv"
)

func withSessionLevelLock(ctx context.Context, name string, client Client) (release func() error, err error) {
	lockID, err := getLockIDByName(name)
	if err != nil {
		return nil, err
	}

	_, err = client.ExecContext(ctx, "select pg_advisory_lock($1)", lockID)
	if err != nil {
		return nil, fmt.Errorf("get lock: %w", err)
	}

	return func() error {
		_, err := client.ExecContext(ctx, "select pg_advisory_unlock($1)", lockID)
		if err != nil {
			return fmt.Errorf("release lock: %w", err)
		}

		return nil
	}, nil
}

func getLockIDByName(name string) (int64, error) {
	hash := fnv.New64a()
	_, err := hash.Write([]byte(name))
	if err != nil {
		return 0, fmt.Errorf("create name hash for lock: %w", err)
	}
	return int64(hash.Sum64()), nil
}
