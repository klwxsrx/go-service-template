package duck

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/internal/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	"github.com/klwxsrx/go-service-template/internal/duck/infra/goose"
	"github.com/klwxsrx/go-service-template/internal/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/duck/infra/sql"
	commonhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkgevent "github.com/klwxsrx/go-service-template/pkg/event"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

const (
	domainName = "duck"
)

type DependencyContainer struct {
	duckService *service.DuckService
}

func MustInitDependencyContainer(
	sqlClient pkgsql.TxClient,
	msgBuses pkgmessage.BusFactory,
	httpClients pkgcmd.HTTPClientFactory,
) *DependencyContainer {
	transaction := pkgsql.NewTransaction(
		sqlClient,
		domainName,
		nil,
	)

	gooseServiceHTTPClient := httpClients.MustInitClient(commonhttp.DestinationGooseService)
	gooseService := goose.NewService(gooseServiceHTTPClient)

	msgBus := msgBuses.New(domainName)
	eventDispatcher := mustInitEventDispatcher(msgBus)

	duckRepo := sql.NewDuckRepo(sqlClient, eventDispatcher)

	return &DependencyContainer{
		duckService: service.NewDuckService(gooseService, transaction, duckRepo),
	}
}

func (c *DependencyContainer) MustRegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(http.NewCreateDuckHandler(c.duckService))
	registry.Register(http.NewSetDuckActiveHandler(c.duckService))
}

func (c *DependencyContainer) MustRegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	err := registry.RegisterHandlers(
		domainName,
		pkgmessage.RegisterEventHandler[domain.EventDuckCreated](domainName, c.duckService.HandleDuckCreated),
		pkgmessage.RegisterEventHandler[external.EventGooseQuacked](goose.DomainName, c.duckService.HandleGooseQuacked),
	)
	if err != nil {
		panic(fmt.Errorf("register %s message handlers: %w", domainName, err))
	}
}

func mustInitEventDispatcher(msgBus pkgmessage.Bus) pkgevent.Dispatcher {
	dispatcher := pkgmessage.NewEventDispatcher(msgBus)
	err := msgBus.RegisterMessages(
		pkgmessage.RegisterEvent[domain.EventDuckCreated](),
	)
	if err != nil {
		panic(fmt.Errorf("register %s events: %w", domainName, err))
	}
	return dispatcher
}
