package main

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
	pkgobservability "github.com/klwxsrx/go-service-template/pkg/observability"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	ctx := context.Background()
	logger := log.New(log.LevelInfo)
	metrics := pkgmetricstub.NewMetrics()
	observability := pkgobservability.New()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	sqlMessageOutbox := pkgcmd.MustInitSQLMessageOutbox(ctx, sqlDB, msgBroker, logger)
	defer sqlMessageOutbox.Close()

	gooseClient := cmd.MustInitGooseHTTPClient(observability, metrics, logger)

	container := pkgduck.NewDependencyContainer(ctx, sqlDB, sqlMessageOutbox, gooseClient)

	messageListenerManager := message.NewListenerManager(
		msgBroker,
		message.WithMetrics(metrics),
		message.WithLogging(logger, log.LevelInfo, log.LevelWarn),
	)
	container.RegisterMessageHandlers(messageListenerManager)

	listenerHub := hub.Run(message.Must(messageListenerManager.Listeners())...)

	logger.Info(ctx, "app is ready")
	hub.Must(listenerHub.Wait(ctx, sig.TermSignals(), logger))
}
