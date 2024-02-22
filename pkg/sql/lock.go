package sql

import (
	"context"
	"fmt"
	"hash/fnv"
)

func withSessionLevelLock(ctx context.Context, name string, client Client) (connCtx context.Context, release func() error, err error) {
	lockID, err := getLockIDByName(name)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancelConn, err := client.WithinSingleConnection(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get connection for %s: %w", name, err)
	}

	_, err = client.ExecContext(ctx, "select pg_advisory_lock($1)", lockID)
	if err != nil {
		cancelConn()
		return nil, nil, fmt.Errorf("get lock for %s: %w", name, err)
	}

	return ctx, func() error {
		defer cancelConn()

		var released bool
		err = client.GetContext(ctx, &released, "select pg_advisory_unlock($1)", lockID)
		if err != nil {
			return fmt.Errorf("release lock for %s: %w", name, err)
		}
		if !released {
			return fmt.Errorf("release lock for %s: lock wasn't released", name)
		}

		return nil
	}, nil
}

func withTransactionLevelLock(ctx context.Context, name string, tx ClientTx) error {
	lockID, err := getLockIDByName(name)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "select pg_advisory_xact_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("get lock for %s: %w", name, err)
	}

	return nil
}

func getLockIDByName(name string) (int64, error) {
	hash := fnv.New64a()
	_, err := hash.Write([]byte(name))
	if err != nil {
		return 0, fmt.Errorf("create hash for lock with name %s: %w", name, err)
	}

	return int64(hash.Sum64()), nil
}
