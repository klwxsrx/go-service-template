package main

import (
	"context"

	sqlduck "github.com/klwxsrx/go-service-template/data/sql/duck"
	"github.com/klwxsrx/go-service-template/internal/duck"
	commonhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	commonmessage "github.com/klwxsrx/go-service-template/internal/pkg/message"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkglog "github.com/klwxsrx/go-service-template/pkg/log"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
	pkgobservability "github.com/klwxsrx/go-service-template/pkg/observability"
	pkgsig "github.com/klwxsrx/go-service-template/pkg/sig"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	logger := pkgcmd.InitLogger()
	metrics := pkgmetricstub.NewMetrics()
	observability := pkgobservability.New(
		pkgobservability.WithFieldsLogging(logger, pkgobservability.LogFieldRequestID),
	)
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, pkgsql.MessageOutboxMigrations, sqlduck.Migrations)
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	msgBuses := commonmessage.NewBusFactory(observability, metrics, logger)

	msgOutbox := pkgcmd.MustInitSQLMessageOutbox(sqlDB, msgBroker, logger)
	defer msgOutbox.Close()

	httpClients := commonhttp.NewClientFactory(observability, metrics, logger)

	container := duck.MustInitDependencyContainer(sqlDB, msgBuses, httpClients, msgOutbox.Process)

	msgBusListener := pkgmessage.NewBusListener(
		msgBroker,
		pkgmessage.WithHandlerObservability(observability),
		pkgmessage.WithHandlerMetrics(metrics),
		pkgmessage.WithHandlerLogging(logger, pkglog.LevelInfo, pkglog.LevelWarn),
	)
	container.MustRegisterMessageHandlers(msgBusListener)

	logger.Info(ctx, "app is ready")
	pkgworker.Must(pkgworker.Run(msgBusListener.ListenerProcesses()...).Wait(ctx, pkgsig.TermSignals(), logger))
}
