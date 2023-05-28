package sql

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/sql"
)

type duckRepo struct {
	db              sql.Client
	eventDispatcher event.Dispatcher
}

func (r *duckRepo) Store(ctx context.Context, duck *domain.Duck) error {
	err := r.eventDispatcher.Dispatch(ctx, duck.Changes)
	if err != nil {
		return fmt.Errorf("failed to dispatch events: %w", err)
	}

	query, args, err := sq.
		Insert("duck").
		Columns("id", "name").
		Values(duck.ID, duck.Name).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build sql: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert duck: %w", err)
	}
	return nil
}

func NewDuckRepo(db sql.Client, dispatcher event.Dispatcher) domain.DuckRepo {
	return &duckRepo{
		db:              db,
		eventDispatcher: dispatcher,
	}
}
