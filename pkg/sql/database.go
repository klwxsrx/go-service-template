package sql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/cenkalti/backoff/v4"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgresql driver

	"github.com/klwxsrx/go-service-template/pkg/log"
)

const defaultConnectionTimeout = 10 * time.Second

type Config struct {
	DSN                DSN
	MaxOpenConnections int
	MaxIdleConnections int
	ConnectionTimeout  time.Duration
}

type DSN struct {
	User     string
	Password string
	Address  string
	Database string
}

func (d *DSN) String() string {
	return fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", d.User, d.Password, d.Address, d.Database)
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
	Close(ctx context.Context)
}

type database struct {
	transactionalClient
	db     *sqlx.DB
	logger log.Logger
}

func NewDatabase(ctx context.Context, config *Config, logger log.Logger) (Database, error) {
	if config.ConnectionTimeout <= 0 {
		config.ConnectionTimeout = defaultConnectionTimeout
	}

	db, err := openConnection(ctx, config)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdleConnections)

	enablePostgreSQLSquirrelPlaceholderFormat()
	return &database{
		transactionalClient: transactionalClient{db},
		db:                  db,
		logger:              logger,
	}, nil
}

func (c *database) Close(ctx context.Context) {
	err := c.db.Close()
	if err != nil {
		c.logger.WithError(err).Error(ctx, "failed to close sql database")
	}
}

func openConnection(ctx context.Context, config *Config) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", config.DSN.String())
	if err != nil {
		return nil, err
	}

	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = time.Second
	eb.RandomizationFactor = 0
	eb.Multiplier = 2
	eb.MaxInterval = config.ConnectionTimeout / 4
	eb.MaxElapsedTime = config.ConnectionTimeout

	err = backoff.Retry(func() error {
		return db.PingContext(ctx)
	}, eb)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

var squirrelPlaceholderOnceDoer = &sync.Once{}

func enablePostgreSQLSquirrelPlaceholderFormat() {
	squirrelPlaceholderOnceDoer.Do(func() {
		sq.StatementBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	})
}
