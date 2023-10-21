package duck

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/sql"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/integration"
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
	msgOutbox pkgmessage.Outbox,
	_ pkghttp.Client,
) *DependencyContainer {
	d := &DependencyContainer{}

	wrappedSQLClient, transaction := pkgsql.NewTransaction(
		sqlClient,
		domainName,
		msgOutbox.Process,
	)

	messageBus := pkgmessage.NewBus(
		domainName,
		pkgsql.NewMessageOutboxStorage(wrappedSQLClient),
	)
	d.mustRegisterMessageBus(messageBus)

	eventDispatcher := pkgmessage.NewEventDispatcher(messageBus)

	duckRepo := sql.NewDuckRepo(wrappedSQLClient, eventDispatcher)
	d.duckService = service.NewDuckService(duckRepo, transaction)

	return d
}

func (d *DependencyContainer) RegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(http.NewCreateDuckHandler(d.duckService))
}

func (d *DependencyContainer) RegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	registry.RegisterHandler(
		domainName, domainName,
		pkgmessage.RegisterEventHandler[domain.EventDuckCreated](d.duckService.HandleDuckCreated),
	)
	registry.RegisterHandler(
		domainName, integration.DomainNameGoose,
		pkgmessage.RegisterEventHandler[external.EventGooseQuacked](d.duckService.HandleGooseQuacked),
	)
}

func (d *DependencyContainer) mustRegisterMessageBus(messageBus pkgmessage.Bus) {
	err := messageBus.RegisterMessage(
		pkgmessage.RegisterEvent[domain.EventDuckCreated](),
	)
	if err != nil {
		panic(fmt.Errorf("failed to register message bus messages: %w", err))
	}
}
