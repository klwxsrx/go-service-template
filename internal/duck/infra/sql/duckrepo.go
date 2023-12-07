package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	pkgevent "github.com/klwxsrx/go-service-template/pkg/event"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type duckRepo struct {
	db              pkgsql.Client
	eventDispatcher pkgevent.Dispatcher
}

func NewDuckRepo(db pkgsql.Client, dispatcher pkgevent.Dispatcher) domain.DuckRepo {
	return &duckRepo{
		db:              db,
		eventDispatcher: dispatcher,
	}
}

func (r *duckRepo) FindOne(ctx context.Context, spec domain.DuckSpec) (*domain.Duck, error) {
	qb := sq.
		Select("id", "name", "is_active").
		From("duck").
		Limit(1)
	if spec.ID != nil {
		qb = qb.Where(sq.Eq{"id": *spec.ID})
	}
	if pkgsql.IsLockRequested(ctx) {
		qb = qb.Suffix("for update")
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql: %w", err)
	}

	var row sqlxDuck
	err = r.db.GetContext(ctx, &row, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrDuckNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get duck: %w", err)
	}

	return &domain.Duck{
		ID:       row.ID,
		Name:     row.Name,
		IsActive: row.IsActive,
		Changes:  nil,
	}, nil
}

func (r *duckRepo) Store(ctx context.Context, duck *domain.Duck) error {
	err := r.eventDispatcher.Dispatch(ctx, duck.Changes...)
	if err != nil {
		return fmt.Errorf("dispatch events: %w", err)
	}

	query, args, err := sq.
		Insert("duck").
		Columns("id", "name", "is_active").
		Values(duck.ID, duck.Name, duck.IsActive).
		Suffix(`on conflict (id) do update set
			name = excluded.name,
			is_active = excluded.is_active,
			updated_at = now()
		`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert duck: %w", err)
	}
	return nil
}

type sqlxDuck struct {
	ID       uuid.UUID `db:"id"`
	Name     string    `db:"name"`
	IsActive bool      `db:"is_active"`
}
