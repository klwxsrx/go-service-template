package duck

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/sql"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/integration"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
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

func NewDependencyContainer(
	ctx context.Context,
	sqlClient pkgsql.TxClient,
	msgOutbox pkgmessage.Outbox,
	_ pkghttp.Client,
) *DependencyContainer {
	d := &DependencyContainer{}

	wrappedSQLClient, transaction := pkgcmd.MustInitSQLTransaction(
		sqlClient,
		domainName,
		msgOutbox.Process,
	)

	messageStore := pkgcmd.MustInitSQLMessageStore(ctx, wrappedSQLClient)
	eventDispatcher := pkgmessage.NewEventDispatcher(
		domainName,
		pkgmessage.NewStoreDispatcher(messageStore),
	)

	duckRepo := sql.NewDuckRepo(wrappedSQLClient, eventDispatcher)
	d.duckService = service.NewDuckService(duckRepo, transaction)

	return d
}

func (d *DependencyContainer) RegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(http.NewCreateDuckHandler(d.duckService))
}

func (d *DependencyContainer) RegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	registry.Register(
		domainName, domainName,
		pkgmessage.RegisterEventHandler[domain.EventDuckCreated](d.duckService.HandleDuckCreated),
	)
	registry.Register(
		domainName, integration.DomainNameGoose,
		pkgmessage.RegisterEventHandler[external.EventGooseQuacked](d.duckService.HandleGooseQuacked),
	)
}
