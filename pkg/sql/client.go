package sql

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Client interface { // TODO: add implementation which handle transactions from ctx
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

type ClientTx interface {
	Client
	Commit() error
	Rollback() error
}

type TxClient interface {
	Client
	Begin(ctx context.Context) (ClientTx, error)
}

type txClient struct {
	*sqlx.DB
}

func (c *txClient) Begin(ctx context.Context) (ClientTx, error) {
	tx, err := c.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
