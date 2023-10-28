package duck

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/sql"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/integration"
	pkgevent "github.com/klwxsrx/go-service-template/pkg/event"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

const (
	domainName = "duck"
)

type DependencyContainer struct {
	duckService service.DuckService
}

func MustInitDependencyContainer(
	sqlClient pkgsql.TxClient,
	_ pkghttp.Client,
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

	eventDispatcher := mustInitEventDispatcher(msgBus)
	duckRepo := sql.NewDuckRepo(wrappedSQLClient, eventDispatcher)

	return &DependencyContainer{
		duckService: service.NewDuckService(transaction, duckRepo),
	}
}

func (c *DependencyContainer) MustRegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(http.NewCreateDuckHandler(c.duckService))
}

func (c *DependencyContainer) MustRegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	err := registry.RegisterHandlers(
		domainName,
		pkgmessage.RegisterEventHandler[domain.EventDuckCreated](domainName, c.duckService.HandleDuckCreated),
		pkgmessage.RegisterEventHandler[external.EventGooseQuacked](integration.DomainNameGoose, c.duckService.HandleGooseQuacked),
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
