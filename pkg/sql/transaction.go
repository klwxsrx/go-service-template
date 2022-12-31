package sql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type transaction struct {
	client   TxClient
	onCommit func()
}

func (t *transaction) Execute(
	ctx context.Context,
	fn func(ctx context.Context) error,
	lockNames ...string,
) error {
	var err error
	tx, isParentTx := ctx.Value(databaseTransactionContextKey).(ClientTx)
	if !isParentTx {
		tx, err = t.client.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start db transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()

		ctx = context.WithValue(ctx, databaseTransactionContextKey, tx)
	}

	for _, lockName := range lockNames {
		err = lockDatabase(ctx, tx, "SELECT pg_advisory_xact_lock($1)", lockName)
		if err != nil {
			return err
		}
	}

	err = fn(ctx)
	if err != nil {
		return err
	}

	if isParentTx {
		return nil
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	t.onCommit()
	return nil
}

func NewTransaction(client TxClient, onCommit func()) (Client, persistence.Transaction) {
	return &txUnwrapperClient{client: client}, &transaction{client: client, onCommit: onCommit}
}

type txUnwrapperClient struct {
	client Client
}

func (c *txUnwrapperClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, ok := ctx.Value(databaseTransactionContextKey).(ClientTx)
	if ok {
		return tx.ExecContext(ctx, query, args...)
	}
	return c.client.ExecContext(ctx, query, args...)
}

func (c *txUnwrapperClient) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	tx, ok := ctx.Value(databaseTransactionContextKey).(ClientTx)
	if ok {
		return tx.NamedExecContext(ctx, query, arg)
	}
	return c.client.NamedExecContext(ctx, query, arg)
}

func (c *txUnwrapperClient) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	tx, ok := ctx.Value(databaseTransactionContextKey).(ClientTx)
	if ok {
		return tx.GetContext(ctx, dest, query, args...)
	}
	return c.client.GetContext(ctx, dest, query, args...)
}

func (c *txUnwrapperClient) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	tx, ok := ctx.Value(databaseTransactionContextKey).(ClientTx)
	if ok {
		return tx.SelectContext(ctx, dest, query, args...)
	}
	return c.client.SelectContext(ctx, dest, query, args...)
}
