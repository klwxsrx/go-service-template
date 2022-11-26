package sql

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const transactionTimeout = 10 * time.Second

type UnitOfWork interface {
	Execute(ctx context.Context, f func(ctx context.Context, dbTx ClientTx) error) error
}

type unitOfWork struct {
	db TxClient
}

func (u *unitOfWork) Execute(ctx context.Context, f func(ctx context.Context, dbTx ClientTx) error) error {
	ctxWithTimeout, cancelFunc := context.WithCancel(ctx)
	time.AfterFunc(transactionTimeout, func() {
		cancelFunc()
	})

	dbTx, err := u.db.Begin(ctxWithTimeout)
	if err != nil {
		return fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer func() {
		_ = dbTx.Rollback()
	}()

	err = f(ctxWithTimeout, dbTx)
	if errors.Is(err, context.Canceled) && ctx.Err() == nil {
		return fmt.Errorf("db transaction timeout exceeded")
	}
	if err != nil {
		return fmt.Errorf("failed to execute in db transaction: %w", err)
	}

	return nil
}

func NewUnitOfWork(db TxClient) UnitOfWork {
	return &unitOfWork{db: db}
}
