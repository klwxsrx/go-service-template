package main

import (
	"context"

	"github.com/klwxsrx/go-service-template/cmd"
	sqlduck "github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
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
	observability := pkgobservability.New()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, pkgsql.MessageOutboxMigrations, sqlduck.Migrations)
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	msgOutbox := pkgcmd.MustInitSQLMessageOutbox(sqlDB, msgBroker, logger)
	defer msgOutbox.Close()

	gooseClient := cmd.MustInitGooseHTTPClient(observability, metrics, logger)

	container := pkgduck.MustInitDependencyContainer(sqlDB, gooseClient, msgOutbox.Process)

	msgBusListener := pkgmessage.NewBusListener(
		msgBroker,
		pkgmessage.WithMetrics(metrics),
		pkgmessage.WithLogging(logger, pkglog.LevelInfo, pkglog.LevelWarn),
	)
	container.MustRegisterMessageHandlers(msgBusListener)

	logger.Info(ctx, "app is ready")
	pkgworker.Must(pkgworker.Run(msgBusListener.ListenerProcesses()...).Wait(ctx, pkgsig.TermSignals(), logger))
}
