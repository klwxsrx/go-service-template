package sql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type (
	instanceID string

	txData struct {
		ClientTx
		instanceID instanceID
	}

	transaction struct {
		id       instanceID
		client   TxClient
		onCommit func()
	}

	updateLock struct {
		forUpdate  bool
		skipLocked bool
	}
)

func NewTransaction(client TxClient, instanceName string, onCommit func()) persistence.Transaction {
	return transaction{id: instanceID(instanceName), client: client, onCommit: onCommit}
}

func (t transaction) WithinContext(
	ctx context.Context,
	fn func(ctx context.Context) error,
	locks ...persistence.Lock,
) error {
	var err error
	storedTx, ok := ctx.Value(dbTransactionContextKey).(txData)
	hasParentTx := ok && storedTx.instanceID == t.id
	if !hasParentTx {
		var tx ClientTx
		conn, hasConn := ctx.Value(dbConnectionContextKey).(*sqlx.Conn)
		if hasConn {
			var txx *sqlx.Tx
			txx, err = conn.BeginTxx(ctx, nil)
			tx = clientTransaction{txx}
		} else {
			tx, err = t.client.Begin(ctx)
		}
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

	slices.SortFunc(locks, func(a, b persistence.Lock) int {
		switch {
		case a.Key < b.Key:
			return -1
		case a.Key > b.Key:
			return 1
		default:
			return 0
		}
	})
	for _, lock := range locks {
		err = withTransactionLevelLock(ctx, lock.Key, lock.Shared, storedTx.ClientTx)
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

func (t transaction) LockUpdate(ctx context.Context, exclusively bool, opts ...persistence.LockUpdateOption) context.Context {
	hasSkipLockedFn := func(opts []persistence.LockUpdateOption) bool {
		for _, opt := range opts {
			if opt == persistence.SkipAlreadyLockedData {
				return true
			}
		}
		return false
	}

	if !HasTransaction(ctx) {
		return ctx
	}

	skipLocked := hasSkipLockedFn(opts)
	lockRequested, hadExclusively, hadSkipLocked := IsUpdateLockRequested(ctx)
	if lockRequested && hadExclusively == exclusively && hadSkipLocked == skipLocked {
		return ctx
	}

	return context.WithValue(ctx, dbTransactionLockContextKey, updateLock{
		forUpdate:  exclusively,
		skipLocked: skipLocked,
	})
}

func HasTransaction(ctx context.Context) bool {
	return ctx.Value(dbTransactionContextKey) != nil
}

func IsUpdateLockRequested(ctx context.Context) (lockRequested, forUpdate, skipLocked bool) {
	lock, ok := ctx.Value(dbTransactionLockContextKey).(updateLock)
	if !ok {
		return
	}

	return true, lock.forUpdate, lock.skipLocked
}

func PrepareUpdateLockQuery(ctx context.Context, qb sq.SelectBuilder) sq.SelectBuilder {
	ok, forUpdate, skipLocked := IsUpdateLockRequested(ctx)
	if !ok {
		return qb
	}

	if forUpdate {
		qb = qb.Suffix("FOR UPDATE")
	} else {
		qb = qb.Suffix("FOR SHARE")
	}
	if skipLocked {
		qb = qb.Suffix("SKIP LOCKED")
	}

	return qb
}

type (
	transactionalClient struct {
		db *sqlx.DB
	}

	clientTransaction struct {
		*sqlx.Tx
	}
)

func (c transactionalClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, ok := ctx.Value(dbTransactionContextKey).(ClientTx)
	if ok {
		return tx.ExecContext(ctx, query, args...)
	}

	conn, ok := ctx.Value(dbConnectionContextKey).(*sqlx.Conn)
	if ok {
		return conn.ExecContext(ctx, query, args...)
	}

	return c.db.ExecContext(ctx, query, args...)
}

func (c transactionalClient) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	tx, ok := ctx.Value(dbTransactionContextKey).(ClientTx)
	if ok {
		return tx.GetContext(ctx, dest, query, args...)
	}

	conn, ok := ctx.Value(dbConnectionContextKey).(*sqlx.Conn)
	if ok {
		return conn.GetContext(ctx, dest, query, args...)
	}

	return c.db.GetContext(ctx, dest, query, args...)
}

func (c transactionalClient) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	tx, ok := ctx.Value(dbTransactionContextKey).(ClientTx)
	if ok {
		return tx.SelectContext(ctx, dest, query, args...)
	}

	conn, ok := ctx.Value(dbConnectionContextKey).(*sqlx.Conn)
	if ok {
		return conn.SelectContext(ctx, dest, query, args...)
	}

	return c.db.SelectContext(ctx, dest, query, args...)
}

func (c transactionalClient) WithinSingleConnection(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if _, ok := ctx.Value(dbConnectionContextKey).(*sqlx.Conn); ok || HasTransaction(ctx) {
		return ctx, func() {}, nil
	}

	conn, err := c.db.Connx(ctx)
	if err != nil {
		return nil, nil, err
	}

	ctx = context.WithValue(ctx, dbConnectionContextKey, conn)
	return ctx, func() { _ = conn.Close() }, nil
}

func (c transactionalClient) Begin(ctx context.Context) (ClientTx, error) {
	type transactional interface {
		BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
	}

	var impl transactional = c.db
	conn, ok := ctx.Value(dbConnectionContextKey).(*sqlx.Conn)
	if ok {
		impl = conn
	}

	tx, err := impl.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return clientTransaction{tx}, nil
}

func (c clientTransaction) WithinSingleConnection(ctx context.Context) (context.Context, context.CancelFunc, error) {
	return ctx, func() {}, nil
}
