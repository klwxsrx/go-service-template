package sql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type instanceID string

type txData struct {
	ClientTx
	instanceID instanceID
}

type transaction struct {
	id       instanceID
	client   TxClient
	onCommit func()
}

func (t *transaction) Execute(
	ctx context.Context,
	fn func(ctx context.Context) error,
	lockNames ...string,
) error {
	var err error
	storedTx, ok := ctx.Value(dbTransactionContextKey).(txData)
	hasParentTx := ok && storedTx.instanceID == t.id
	if !hasParentTx {
		var tx ClientTx
		tx, err = t.client.Begin(ctx)
		if err != nil {
			return fmt.Errorf("start db transaction: %w", err)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()

		storedTx.instanceID = t.id
		storedTx.ClientTx = tx
		ctx = context.WithValue(ctx, dbTransactionContextKey, storedTx)
	}

	for _, lockName := range lockNames {
		err = lockDatabase(ctx, storedTx.ClientTx, "SELECT pg_advisory_xact_lock($1)", lockName)
		if err != nil {
			return err
		}
	}

	err = fn(ctx)
	if err != nil {
		return err
	}

	if hasParentTx {
		return nil
	}

	err = storedTx.ClientTx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	if t.onCommit != nil {
		t.onCommit()
	}

	return nil
}

func NewTransaction(client TxClient, instanceName string, onCommit func()) persistence.Transaction {
	return &transaction{id: instanceID(instanceName), client: client, onCommit: onCommit}
}

type transactionalClient struct {
	client Client
}

func (c *transactionalClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, ok := ctx.Value(dbTransactionContextKey).(Client)
	if ok {
		return tx.ExecContext(ctx, query, args...)
	}
	return c.client.ExecContext(ctx, query, args...)
}

func (c *transactionalClient) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	tx, ok := ctx.Value(dbTransactionContextKey).(Client)
	if ok {
		return tx.NamedExecContext(ctx, query, arg)
	}
	return c.client.NamedExecContext(ctx, query, arg)
}

func (c *transactionalClient) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	tx, ok := ctx.Value(dbTransactionContextKey).(Client)
	if ok {
		return tx.GetContext(ctx, dest, query, args...)
	}
	return c.client.GetContext(ctx, dest, query, args...)
}

func (c *transactionalClient) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	tx, ok := ctx.Value(dbTransactionContextKey).(Client)
	if ok {
		return tx.SelectContext(ctx, dest, query, args...)
	}
	return c.client.SelectContext(ctx, dest, query, args...)
}
