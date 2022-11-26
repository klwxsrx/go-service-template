package sql

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/jmoiron/sqlx"
	"github.com/klwxsrx/go-service-template/pkg/log"
	_ "github.com/lib/pq" // postgres driver
	"time"
)

const defaultConnectionTimeout = 20 * time.Second

type Config struct {
	DSN               DSN
	ConnectionTimeout time.Duration
}

type DSN struct {
	User     string
	Password string
	Address  string
	Database string
}

func (d *DSN) String() string {
	return fmt.Sprintf("postgresql://%s:%s@%s/%s", d.User, d.Password, d.Address, d.Database)
}

type Connection interface {
	Client() TxClient
	Close(ctx context.Context)
}

type connection struct {
	db     *sqlx.DB
	logger log.Logger
}

func (c *connection) Client() TxClient {
	return &txClient{c.db}
}

func (c *connection) Close(ctx context.Context) {
	err := c.db.Close()
	if err != nil {
		c.logger.WithError(err).Error(ctx, "failed to close sql connection")
	}
}

func openConnection(config *Config) (*sqlx.DB, error) {
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
		return db.Ping()
	}, eb)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to open sql connection: %w", err)
	}
	return db, nil
}

func NewConnection(config *Config, logger log.Logger) (Connection, error) {
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = defaultConnectionTimeout
	}

	db, err := openConnection(config)
	if err != nil {
		return nil, err
	}

	return &connection{
		db:     db,
		logger: logger,
	}, nil
}
