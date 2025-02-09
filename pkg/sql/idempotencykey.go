package sql

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/idk"
)

type IdempotencyKeyStorage struct {
	db Client
}

func NewIdempotencyKeyStorage(db Client) idk.Storage {
	return IdempotencyKeyStorage{db: db}
}

func (s IdempotencyKeyStorage) Insert(ctx context.Context, key uuid.UUID, extraKey string) error {
	query, args, err := sq.
		Insert("idempotency_key").
		Columns("key", "extra_key").
		Values(key, extraKey).
		Suffix("on conflict do nothing").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return idk.ErrAlreadyInserted
	}

	return nil
}

func (s IdempotencyKeyStorage) Delete(ctx context.Context, createdAtBefore time.Time) error {
	query, args, err := sq.
		Delete("idempotency_key").
		Where(sq.Lt{"created_at": createdAtBefore}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete keys: %w", err)
	}

	return nil
}

func IdempotencyKeyMigrations() ([]Migration, error) {
	return []Migration{
		{
			ID: "0000-00-00-001-create-idempotency-key-table",
			SQL: `
				create table if not exists idempotency_key (
					key        uuid        not null,
					extra_key  text        not null,
					created_at timestamptz not null default current_timestamp,
					primary key (key, extra_key)
				);
			`,
		},
	}, nil
}
