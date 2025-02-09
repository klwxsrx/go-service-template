package infra

import (
	"github.com/klwxsrx/go-service-template/data/sql/user"
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	"github.com/klwxsrx/go-service-template/internal/user/infra/sql"
	userinfrasqlgenerated "github.com/klwxsrx/go-service-template/internal/user/infra/sql/generated"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/lazy"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type SQLContainer struct {
	UserRepo lazy.Loader[domain.UserRepository]
}

func NewSQLContainer(
	db lazy.Loader[pkgsql.Database],
	dbMigrations lazy.Loader[cmd.SQLMigrations],
	eventDispatcher lazy.Loader[event.Dispatcher],
) lazy.Loader[SQLContainer] {
	return lazy.New(func() (SQLContainer, error) {
		dbMigrations.MustLoad().MustRegister(user.Migrations)

		sqlxConverter := sqlxConverterProvider()
		return SQLContainer{
			UserRepo: userRepoProvider(db, eventDispatcher, sqlxConverter),
		}, nil
	})
}

func sqlxConverterProvider() lazy.Loader[sql.SqlxConverter] {
	return lazy.New(func() (sql.SqlxConverter, error) {
		return &userinfrasqlgenerated.SqlxConverterImpl{}, nil
	})
}

func userRepoProvider(
	db lazy.Loader[pkgsql.Database],
	eventDispatcher lazy.Loader[event.Dispatcher],
	sqlxConverter lazy.Loader[sql.SqlxConverter],
) lazy.Loader[domain.UserRepository] {
	return lazy.New(func() (domain.UserRepository, error) {
		return sql.NewUserRepository(db.MustLoad(), eventDispatcher.MustLoad(), sqlxConverter.MustLoad()), nil
	})
}
