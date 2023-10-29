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
	httpClients *commonhttp.ClientFactory,
	onCommit func(),
) *DependencyContainer {
	wrappedSQLClient, transaction := pkgsql.NewTransaction(
		sqlClient,
		domainName,
		onCommit,
	)

	msgBus := pkgmessage.NewBus(
		domainName,
		pkgsql.NewMessageOutboxStorage(wrappedSQLClient),
	)

	gooseServiceHTTPClient := httpClients.MustInitClient(commonhttp.DestinationGooseService)
	gooseService := goose.NewService(gooseServiceHTTPClient)

	eventDispatcher := mustInitEventDispatcher(msgBus)
	duckRepo := sql.NewDuckRepo(wrappedSQLClient, eventDispatcher)

	return &DependencyContainer{
		duckService: service.NewDuckService(gooseService, transaction, duckRepo),
	}
}

func (c *DependencyContainer) MustRegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(http.NewCreateDuckHandler(c.duckService))
}

func (c *DependencyContainer) MustRegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	err := registry.RegisterHandlers(
		domainName,
		pkgmessage.RegisterEventHandler[domain.EventDuckCreated](domainName, c.duckService.HandleDuckCreated),
		pkgmessage.RegisterEventHandler[external.EventGooseQuacked](goose.DomainName, c.duckService.HandleGooseQuacked),
	)
	if err != nil {
		panic(fmt.Errorf("failed to register %s message handlers: %w", domainName, err))
	}
}

func mustInitEventDispatcher(msgBus pkgmessage.Bus) pkgevent.Dispatcher {
	dispatcher := pkgmessage.NewEventDispatcher(msgBus)
	err := msgBus.RegisterMessages(
		pkgmessage.RegisterEvent[domain.EventDuckCreated](),
	)
	if err != nil {
		panic(fmt.Errorf("failed to register %s events: %w", domainName, err))
	}
	return dispatcher
}