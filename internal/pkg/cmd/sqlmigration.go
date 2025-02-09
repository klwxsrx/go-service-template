package cmd

import (
	"context"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/sql"
)

type (
	SQLMigrations interface {
		MustRegister(sources ...sql.MigrationSource)
	}

	sqlMigrations struct {
		ctx    context.Context
		db     sql.Database
		logger log.Logger
	}
)

func NewSQLMigrations(
	ctx context.Context,
	db sql.Database,
	logger log.Logger,
) SQLMigrations {
	return &sqlMigrations{
		ctx:    ctx,
		db:     db,
		logger: logger,
	}
}

func (s *sqlMigrations) MustRegister(sources ...sql.MigrationSource) {
	if len(sources) == 0 {
		return
	}

	err := sql.NewMigrator(s.db, s.logger).Execute(s.ctx, sources...)
	if err != nil {
		panic(fmt.Errorf("execute migrations: %w", err))
	}
}
