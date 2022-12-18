package sql

import (
	"context"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"io/fs"
	"strings"
)

const (
	migrationLock  = "perform_migration_lock"
	querySeparator = ";\n"

	migrationTableDDL = `
		CREATE TABLE IF NOT EXISTS migration (
			id text PRIMARY KEY
		)
	`
)

type Migration struct {
	txClient   TxClient
	migrations fs.ReadDirFS
	logger     log.Logger
}

func (m *Migration) Execute(ctx context.Context) error {
	lock := newLock(ctx, migrationLock, m.txClient)

	err := lock.Get()
	if err != nil {
		return fmt.Errorf("failed to get migration lock: %w", err)
	}
	defer lock.Release()

	err = m.createMigrationTableIfNotExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	return m.performFileMigrations(ctx)
}

func (m *Migration) performFileMigrations(ctx context.Context) error {
	migrationIDs, err := m.getFileNames()
	if err != nil {
		return fmt.Errorf("failed to get migration file names: %w", err)
	}
	if len(migrationIDs) == 0 {
		return nil
	}

	performedMigrationIDs, err := m.getPerformedMigrationIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get performed migrations: %w", err)
	}

	for _, migrationID := range migrationIDs {
		if _, ok := performedMigrationIDs[migrationID]; ok {
			continue
		}

		migrationSQL, err := m.readFile(migrationID)
		if err != nil {
			return fmt.Errorf("failed to read migration sql: %w", err)
		}

		err = m.performMigration(ctx, migrationID, migrationSQL)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) getFileNames() ([]string, error) {
	entries, err := m.migrations.ReadDir(".")
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		result = append(result, entry.Name())
	}
	return result, nil
}

func (m *Migration) readFile(fileName string) (string, error) {
	content, err := fs.ReadFile(m.migrations, fileName)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (m *Migration) performMigration(ctx context.Context, migrationID, migrationSQL string) error {
	tx, err := m.txClient.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start tx: %w", err)
	}

	err = m.processMigration(ctx, tx, migrationID, migrationSQL)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration %s failed: %w", migrationID, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	m.logger.WithField("migrationID", migrationID).Info(ctx, "migration executed successfully")
	return nil
}

func (m *Migration) processMigration(ctx context.Context, client Client, migrationID, migrationSQL string) error {
	if migrationSQL == "" {
		return errors.New("empty migration")
	}

	err := m.createMigrationRecord(ctx, client, migrationID)
	if err != nil {
		return err
	}

	queries := m.splitToQueries(migrationSQL)
	for _, query := range queries {
		_, err = client.ExecContext(ctx, query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) createMigrationTableIfNotExists(ctx context.Context) error {
	_, err := m.txClient.ExecContext(ctx, migrationTableDDL)
	return err
}

func (m *Migration) getPerformedMigrationIDs(ctx context.Context) (map[string]struct{}, error) {
	var fileNames []string
	err := m.txClient.SelectContext(ctx, &fileNames, `SELECT id FROM migration`)
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(fileNames))
	for _, id := range fileNames {
		result[id] = struct{}{}
	}
	return result, nil
}

func (m *Migration) createMigrationRecord(ctx context.Context, client Client, fileName string) error {
	_, err := client.ExecContext(ctx, `INSERT INTO migration VALUES ($1)`, fileName)
	return err
}

func (m *Migration) splitToQueries(sql string) []string {
	return strings.Split(sql, querySeparator)
}

func NewMigration(txClient TxClient, migrations fs.ReadDirFS, logger log.Logger) *Migration {
	return &Migration{txClient, migrations, logger}
}
