package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgresql driver
)

type (
	DSN struct {
		User             string
		Password         string
		Host             string
		Port             int
		Database         string
		AdditionalParams map[string]string
	}

	Option func(*sqlx.DB)
)

func (d *DSN) String() string {
	credentials := d.User
	if d.Password != "" {
		credentials = fmt.Sprintf("%s:%s", d.User, d.Password)
	}

	address := d.Host
	if d.Port > 0 {
		address = fmt.Sprintf("%s:%d", d.Host, d.Port)
	}

	params := strings.Builder{}
	if len(d.AdditionalParams) > 0 {
		params.WriteRune('?')
		for key, value := range d.AdditionalParams {
			params.WriteString(fmt.Sprintf("%s=%s&", key, value))
		}
	}

	return fmt.Sprintf("postgresql://%s@%s/%s%s", credentials, address, d.Database, params.String())
}

type Client interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	WithinSingleConnection(ctx context.Context) (context.Context, context.CancelFunc, error)
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

type Database interface {
	TxClient
	Close() error
}

type database struct {
	transactionalClient
	db *sqlx.DB
}

func NewDatabase(dsn DSN, opts ...Option) (Database, error) {
	db, err := sqlx.Open("postgres", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open postgresql connection: %w", err)
	}

	for _, opt := range opts {
		opt(db)
	}

	enablePostgreSQLSquirrelPlaceholderFormat()
	return &database{
		transactionalClient: transactionalClient{db},
		db:                  db,
	}, nil
}

func (c *database) Close() error {
	return c.db.Close()
}

func WithMaxOpenConnections(n int) Option {
	return func(db *sqlx.DB) {
		db.SetMaxOpenConns(n)
	}
}

func WithMaxIdleConnections(n int) Option {
	return func(db *sqlx.DB) {
		db.SetMaxIdleConns(n)
	}
}

func WithMaxConnectionLifetime(d time.Duration) Option {
	return func(db *sqlx.DB) {
		db.SetConnMaxLifetime(d)
	}
}

func WithMaxIdleConnectionTime(d time.Duration) Option {
	return func(db *sqlx.DB) {
		db.SetConnMaxIdleTime(d)
	}
}

var squirrelPlaceholderSetOnce = &sync.Once{}

func enablePostgreSQLSquirrelPlaceholderFormat() {
	squirrelPlaceholderSetOnce.Do(func() {
		sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	})
}
