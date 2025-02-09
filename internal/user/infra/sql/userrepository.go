package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/user/domain"
	"github.com/klwxsrx/go-service-template/pkg/event"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type userRepository struct {
	db              pkgsql.Client
	eventDispatcher event.Dispatcher
	converter       SqlxConverter
}

func NewUserRepository(
	db pkgsql.Client,
	eventDispatcher event.Dispatcher,
	converter SqlxConverter,
) domain.UserRepository {
	return userRepository{converter: converter, db: db, eventDispatcher: eventDispatcher}
}

func (r userRepository) NextID() domain.UserID {
	return domain.UserID{UUID: uuid.New()}
}

func (r userRepository) Store(ctx context.Context, user *domain.User) error {
	err := r.eventDispatcher.Dispatch(ctx, user.Changes...)
	if err != nil {
		return fmt.Errorf("dispatch events: %w", err)
	}

	query, args, err := sq.
		Insert("\"user\"").
		Columns("id", "login", "password_hash", "deleted_at").
		Values(user.ID, user.Login, user.PasswordHash, user.DeletedAt).
		Suffix(`on conflict (id) do update set
			login = excluded.login,
			password_hash = excluded.password_hash,
			deleted_at = excluded.deleted_at,
			updated_at = now()
		`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r userRepository) Find(ctx context.Context, spec domain.FindUserSpecification) ([]domain.User, error) {
	qb := r.buildFindQuery(ctx, spec)
	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var rows []SqlxUser
	err = r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}

	return r.converter.ToDomainUsers(rows), nil
}

func (r userRepository) FindOne(ctx context.Context, spec domain.FindUserSpecification) (*domain.User, error) {
	qb := r.buildFindQuery(ctx, spec).Limit(1)
	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var row SqlxUser
	err = r.db.GetContext(ctx, &row, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return r.converter.ToDomainUser(&row), nil
}

func (r userRepository) buildFindQuery(ctx context.Context, spec domain.FindUserSpecification) sq.SelectBuilder {
	qb := sq.
		Select("id", "login", "password_hash", "deleted_at").
		From("\"user\"")
	if len(spec.IDs) > 0 {
		qb = qb.Where(sq.Eq{"id": spec.IDs})
	}
	if len(spec.Logins) > 0 {
		qb = qb.Where(sq.Eq{"login": spec.Logins})
	}
	if ok, _ := pkgsql.IsLockRequested(ctx); ok {
		qb = qb.Suffix("for update")
	}

	return qb
}

type SqlxUser struct {
	ID           domain.UserID `db:"id"`
	Login        string        `db:"login"`
	PasswordHash string        `db:"password_hash"`
	DeletedAt    *time.Time    `db:"deleted_at"`
}
