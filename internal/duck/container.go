package duck

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/internal/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	"github.com/klwxsrx/go-service-template/internal/duck/infra/goose"
	"github.com/klwxsrx/go-service-template/internal/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/duck/infra/sql"
	commoncmd "github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	commonhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	pkgevent "github.com/klwxsrx/go-service-template/pkg/event"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	pkglazy "github.com/klwxsrx/go-service-template/pkg/lazy"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgpersistence "github.com/klwxsrx/go-service-template/pkg/persistence"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

const (
	domainName = "duck"
)

type DependencyContainer struct {
	DuckService          pkglazy.Loader[*service.DuckService]
	CreateDuckHandler    pkglazy.Loader[http.CreateDuckHandler]
	SetDuckActiveHandler pkglazy.Loader[http.SetDuckActiveHandler]
}

func NewDependencyContainer(
	db pkglazy.Loader[pkgsql.Database],
	dbMigrations pkglazy.Loader[commoncmd.SQLMigrations],
	msgBusProducer pkglazy.Loader[pkgmessage.BusProducer],
	httpClients pkglazy.Loader[commoncmd.HTTPClientFactory],
) *DependencyContainer {
	eventDispatcher := eventDispatcherProvider(msgBusProducer)

	transaction := transactionProvider(db)
	sqlContainer := sql.NewDependencyContainer(db, dbMigrations, eventDispatcher)

	gooseService := gooseServiceProvider(httpClients)

	duckService := duckServiceProvider(
		gooseService,
		transaction,
		sqlContainer,
	)

	return &DependencyContainer{
		DuckService: duckService,
		CreateDuckHandler: pkglazy.New(func() (http.CreateDuckHandler, error) {
			return http.NewCreateDuckHandler(duckService.MustLoad()), nil
		}),
		SetDuckActiveHandler: pkglazy.New(func() (http.SetDuckActiveHandler, error) {
			return http.NewSetDuckActiveHandler(duckService.MustLoad()), nil
		}),
	}
}

func (c *DependencyContainer) MustRegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(c.CreateDuckHandler.MustLoad())
	registry.Register(c.SetDuckActiveHandler.MustLoad())
}

func (c *DependencyContainer) MustRegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	err := registry.RegisterHandlers(
		domainName,
		pkgmessage.RegisterEventHandler[domain.EventDuckCreated](domainName, c.DuckService.MustLoad().HandleDuckCreated),
		pkgmessage.RegisterEventHandler[external.EventGooseQuacked](goose.DomainName, c.DuckService.MustLoad().HandleGooseQuacked),
	)
	if err != nil {
		panic(fmt.Errorf("register %s message handlers: %w", domainName, err))
	}
}

func eventDispatcherProvider(
	msgBus pkglazy.Loader[pkgmessage.BusProducer],
) pkglazy.Loader[pkgevent.Dispatcher] {
	return pkglazy.New(func() (pkgevent.Dispatcher, error) {
		err := msgBus.MustLoad().RegisterMessages(
			domainName,
			pkgmessage.RegisterEvent[domain.EventDuckCreated](),
		)
		if err != nil {
			panic(fmt.Errorf("register %s events: %w", domainName, err))
		}

		return pkgmessage.NewEventDispatcher(domainName, msgBus.MustLoad()), nil
	})
}

func transactionProvider(
	db pkglazy.Loader[pkgsql.Database],
) pkglazy.Loader[pkgpersistence.Transaction] {
	return pkglazy.New(func() (pkgpersistence.Transaction, error) {
		return pkgsql.NewTransaction(
			db.MustLoad(),
			domainName,
			nil,
		), nil
	})
}

func gooseServiceProvider(
	httpClients pkglazy.Loader[commoncmd.HTTPClientFactory],
) pkglazy.Loader[external.GooseService] {
	return pkglazy.New(func() (external.GooseService, error) {
		return goose.NewService(
			httpClients.MustLoad().MustInitClient(commonhttp.DestinationGooseService),
		), nil
	})
}

func duckServiceProvider(
	gooseService pkglazy.Loader[external.GooseService],
	transaction pkglazy.Loader[pkgpersistence.Transaction],
	sqlContainer pkglazy.Loader[*sql.DependencyContainer],
) pkglazy.Loader[*service.DuckService] {
	return pkglazy.New(func() (*service.DuckService, error) {
		return service.NewDuckService(
			gooseService.MustLoad(),
			transaction.MustLoad(),
			sqlContainer.MustLoad().DuckRepo.MustLoad(),
		), nil
	})
}
