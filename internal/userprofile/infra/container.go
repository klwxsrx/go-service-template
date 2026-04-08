package infra

import (
	"github.com/klwxsrx/go-service-template/data/sql/userprofile"
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	"github.com/klwxsrx/go-service-template/internal/userprofile/infra/sql"
	userprofileinfrasqlgenerated "github.com/klwxsrx/go-service-template/internal/userprofile/infra/sql/generated"
	"github.com/klwxsrx/go-service-template/pkg/lazy"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type SQLContainer struct {
	UserProfileRepo lazy.Loader[domain.UserProfileRepository]
}

func NewSQLContainer(
	db lazy.Loader[pkgsql.Database],
	dbMigrations lazy.Loader[cmd.SQLMigrations],
) func() (SQLContainer, error) {
	return func() (SQLContainer, error) {
		dbMigrations.MustLoad().MustRegister(userprofile.Migrations)

		sqlxConverter := lazy.New(sqlxConverterProvider)
		return SQLContainer{
			UserProfileRepo: lazy.New(userProfileRepoProvider(db, sqlxConverter)),
		}, nil
	}
}

func sqlxConverterProvider() (sql.SqlxConverter, error) {
	return &userprofileinfrasqlgenerated.SqlxConverterImpl{}, nil
}

func userProfileRepoProvider(
	db lazy.Loader[pkgsql.Database],
	sqlxConverter lazy.Loader[sql.SqlxConverter],
) func() (domain.UserProfileRepository, error) {
	return func() (domain.UserProfileRepository, error) {
		return sql.NewUserProfileRepository(db.MustLoad(), sqlxConverter.MustLoad()), nil
	}
}
