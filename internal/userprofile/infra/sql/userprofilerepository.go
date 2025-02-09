package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type userProfileRepository struct {
	db        pkgsql.Client
	converter SqlxConverter
}

func NewUserProfileRepository(
	db pkgsql.Client,
	converter SqlxConverter,
) domain.UserProfileRepository {
	return userProfileRepository{converter: converter, db: db}
}

func (r userProfileRepository) Store(ctx context.Context, userProfile *domain.UserProfile) error {
	query, args, err := sq.
		Insert("user_profile").
		Columns("user_id", "first_name", "last_name").
		Values(userProfile.ID, userProfile.FirstName, userProfile.LastName).
		Suffix(`on conflict (user_id) do update set
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			updated_at = now()
		`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r userProfileRepository) FindByID(ctx context.Context, userID domain.UserID) (*domain.UserProfile, error) {
	query, args, err := sq.
		Select("user_id", "first_name", "last_name").
		From("user_profile").
		Where(sq.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var row SqlxUserProfile
	err = r.db.GetContext(ctx, &row, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserProfileNotFound
	}
	if err != nil {
		return nil, err
	}

	return r.converter.ToDomainUserProfile(&row), nil
}

func (r userProfileRepository) DeleteByID(ctx context.Context, userID domain.UserID) error {
	query, args, err := sq.Delete("user_profile").Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrUserProfileNotFound
	}

	return nil
}

type SqlxUserProfile struct {
	ID        domain.UserID `db:"user_id"`
	FirstName string        `db:"first_name"`
	LastName  string        `db:"last_name"`
}
