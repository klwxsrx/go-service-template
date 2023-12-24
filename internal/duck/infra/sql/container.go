package sql

import (
	sqlduck "github.com/klwxsrx/go-service-template/data/sql/duck"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	commoncmd "github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgevent "github.com/klwxsrx/go-service-template/pkg/event"
	pkglazy "github.com/klwxsrx/go-service-template/pkg/lazy"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type DependencyContainer struct {
	DuckRepo pkglazy.Loader[domain.DuckRepo]
}

func NewDependencyContainer(
	db pkglazy.Loader[pkgsql.Database],
	dbMigrations pkglazy.Loader[commoncmd.SQLMigrations],
	eventDispatcher pkglazy.Loader[pkgevent.Dispatcher],
) pkglazy.Loader[*DependencyContainer] {
	return pkglazy.New(func() (*DependencyContainer, error) {
		dbMigrations.MustLoad().MustRegisterSource(sqlduck.Migrations)
		return &DependencyContainer{
			DuckRepo: duckRepoProvider(db, eventDispatcher),
		}, nil
	})
}

func duckRepoProvider(
	db pkglazy.Loader[pkgsql.Database],
	eventDispatcher pkglazy.Loader[pkgevent.Dispatcher],
) pkglazy.Loader[domain.DuckRepo] {
	return pkglazy.New(func() (domain.DuckRepo, error) {
		return NewDuckRepo(db.MustLoad(), eventDispatcher.MustLoad()), nil
	})
}
