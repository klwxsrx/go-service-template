package sql

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

const (
	migrationLockName               = "perform_migration"
	querySeparator                  = ";\n"
	pqErrorCodeRelationDoesNotExist = "42P01"
)

type (
	Migration struct {
		ID  string
		SQL string
	}

	MigrationSource func() ([]Migration, error)
)

type Migrator struct {
	txClient TxClient
	logger   log.Logger
}

func NewMigrator(txClient TxClient, logger log.Logger) *Migrator {
	return &Migrator{
		txClient: txClient,
		logger:   logger,
	}
}

func (m *Migrator) Execute(ctx context.Context, migrationSources ...MigrationSource) error {
	if len(migrationSources) == 0 {
		return nil
	}

	migrationSources = append(migrationSources, migrationTableDDL)

	var migrations []Migration
	for _, migrationSource := range migrationSources {
		sourceMigrations, err := migrationSource()
		if err != nil {
			return fmt.Errorf("get migrations from source: %w", err)
		}
		migrations = append(migrations, sourceMigrations...)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	ctx, releaseLock, err := withSessionLevelLock(ctx, migrationLockName, m.txClient)
	if err != nil {
		return fmt.Errorf("get migration lock: %w", err)
	}
	defer func() {
		err = releaseLock()
		if err != nil {
			m.logger.WithError(err).Error(ctx, "failed to release migration lock")
		}
	}()

	return m.performMigrations(ctx, migrations)
}

func (m *Migrator) performMigrations(ctx context.Context, migrations []Migration) error {
	performedMigrationIDs, err := m.getPerformedMigrationIDs(ctx)
	if err != nil {
		return fmt.Errorf("get performed migrations: %w", err)
	}

	for _, migration := range migrations {
		if _, ok := performedMigrationIDs[migration.ID]; ok {
			continue
		}

		err = m.performMigration(ctx, migration)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) getPerformedMigrationIDs(ctx context.Context) (map[string]struct{}, error) {
	query, _, err := sq.Select("id").From("migration").ToSql()
	if err != nil {
		return nil, err
	}

	var pqErr *pq.Error
	var fileNames []string
	err = m.txClient.SelectContext(ctx, &fileNames, query)
	if errors.As(err, &pqErr) && pqErr.Code == pqErrorCodeRelationDoesNotExist {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := make(map[string]struct{}, len(fileNames))
	for _, id := range fileNames {
		result[id] = struct{}{}
	}

	return result, nil
}

func (m *Migrator) performMigration(ctx context.Context, migration Migration) error {
	tx, err := m.txClient.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start tx: %w", err)
	}

	err = m.performMigrationImpl(ctx, tx, migration)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration %s failed: %w", migration.ID, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	m.logger.WithField("migrationID", migration.ID).Info(ctx, "migration executed successfully")
	return nil
}

func (m *Migrator) performMigrationImpl(ctx context.Context, client Client, migration Migration) error {
	migration.SQL = strings.TrimSpace(migration.SQL)
	if migration.SQL == "" {
		return errors.New("empty migration")
	}

	var err error
	queries := m.splitIntoQueries(migration.SQL)
	for _, query := range queries {
		_, err = client.ExecContext(ctx, query)
		if err != nil {
			return err
		}
	}

	return m.createMigrationRecord(ctx, client, migration.ID)
}

func (m *Migrator) splitIntoQueries(sql string) []string {
	queries := strings.Split(sql, querySeparator)
	result := make([]string, 0, len(queries))
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query != "" {
			result = append(result, query)
		}
	}
	return result
}

func (m *Migrator) createMigrationRecord(ctx context.Context, client Client, fileName string) error {
	query, args, err := sq.Insert("migration").Values(fileName).ToSql()
	if err != nil {
		return err
	}

	_, err = client.ExecContext(ctx, query, args...)
	return err
}

func FSMigrations(fsys fs.ReadDirFS) MigrationSource {
	return func() ([]Migration, error) {
		entries, err := fsys.ReadDir(".")
		if err != nil {
			return nil, fmt.Errorf("read filesystem: %w", err)
		}

		result := make([]Migration, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			fileName := entry.Name()
			fileContent, err := fs.ReadFile(fsys, entry.Name())
			if err != nil {
				return nil, fmt.Errorf("read %s file", entry.Name())
			}

			result = append(result, Migration{
				ID:  strings.TrimSuffix(fileName, filepath.Ext(fileName)),
				SQL: string(fileContent),
			})
		}

		return result, nil
	}
}

func migrationTableDDL() ([]Migration, error) {
	return []Migration{
		{
			ID: "0000-00-00-000-create-migration-table",
			SQL: `
				create table if not exists migration (
					id text primary key
				)
			`,
		},
	}, nil
}
