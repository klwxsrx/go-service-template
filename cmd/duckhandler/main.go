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
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	logger := pkgcmd.InitLogger()
	metrics := pkgmetricstub.NewMetrics()
	observability := pkgobservability.New()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, sqlduck.Migrations)
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	sqlMessageOutbox := pkgcmd.MustInitSQLMessageOutbox(ctx, sqlDB, msgBroker, logger)
	defer sqlMessageOutbox.Close()

	gooseClient := cmd.MustInitGooseHTTPClient(observability, metrics, logger)

	container := pkgduck.NewDependencyContainer(ctx, sqlDB, sqlMessageOutbox, gooseClient)

	messageListenerManager := pkgmessage.NewListenerManager(
		msgBroker,
		pkgmessage.NewDefaultPanicHandler(
			pkgmessage.WithPanicMetrics(metrics),
			pkgmessage.WithPanicLogging(logger),
		),
		pkgmessage.WithMetrics(metrics),
		pkgmessage.WithLogging(logger, pkglog.LevelInfo, pkglog.LevelWarn),
	)
	container.RegisterMessageHandlers(messageListenerManager)

	hub := pkgworker.RunHub(pkgmessage.Must(messageListenerManager.Listeners())...)

	logger.Info(ctx, "app is ready")
	pkgworker.Must(hub.Wait(ctx, pkgsig.TermSignals(), logger))
}
