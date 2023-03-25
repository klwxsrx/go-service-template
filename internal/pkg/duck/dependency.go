package duck

import (
	"context"
	duckappmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	duckinfrasql "github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/sql"
	"github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
)

const (
	MessageSubscriberServiceName = "duck-service"

	moduleName = "duck"
)

type DependencyContainer struct {
	sqlMessageOutbox message.Outbox
	duckService      service.DuckService
}

func NewDependencyContainer(
	ctx context.Context,
	sqlConn sql.Connection,
	pulsarConn pulsar.Connection,
	_ http.Client,
	logger log.Logger,
) *DependencyContainer {
	d := &DependencyContainer{}
	d.sqlMessageOutbox = cmd.MustInitSQLMessageOutbox(ctx, sqlConn, pulsarConn, logger)

	sqlClient, transaction := cmd.MustInitSQLTransaction(
		sqlConn,
		moduleName,
		func() {
			d.sqlMessageOutbox.Process()
		})

	messageStore := cmd.MustInitSQLMessageStore(ctx, sqlClient)
	eventDispatcher := message.NewEventDispatcher(
		message.NewStoreSender(messageStore),
		message.NewJSONEventSerializer(duckappmessage.DuckDomainEventTopicName),
	)

	duckRepo := duckinfrasql.NewDuckRepo(sqlClient, eventDispatcher)
	d.duckService = service.NewDuckService(duckRepo, transaction)

	return d
}

func (d *DependencyContainer) DuckService() service.DuckService {
	return d.duckService
}

func (d *DependencyContainer) Close() {
	d.sqlMessageOutbox.Close()
}
