package main

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/external"
	duckappmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	duckintegrationmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/integration/message"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/event"
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

	sqlConn := pkgcmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlConn.Close(ctx)

	pulsarConn := pkgcmd.MustInitPulsar(nil)
	defer pulsarConn.Close()

	gooseClient := cmd.MustInitGooseHTTPClient(observability, metrics, logger)

	container := pkgduck.NewDependencyContainer(ctx, sqlConn.Client(), pulsarConn.Producer(), gooseClient, logger)
	defer container.Close()

	duckTopicConsumer := pkgcmd.MustInitPulsarFailoverConsumer(pulsarConn, duckappmessage.DuckDomainEventTopicName, pkgduck.MessageSubscriberServiceName)
	defer duckTopicConsumer.Close()

	gooseTopicConsumer := pkgcmd.MustInitPulsarFailoverConsumer(pulsarConn, duckintegrationmessage.GooseDomainEventTopicName, pkgduck.MessageSubscriberServiceName)
	defer gooseTopicConsumer.Close()

	duckService := container.DuckService()
	duckEventMessageHandler := message.NewEventHandler(duckappmessage.NewEventDeserializer(), message.EventTypeHandlerMap{
		domain.EventTypeDuckCreated: event.NewTypedHandler[domain.EventDuckCreated](duckService.HandleDuckCreated),
	})
	gooseEventMessageHandler := message.NewEventHandler(duckintegrationmessage.NewGooseEventDeserializer(), message.EventTypeHandlerMap{
		external.EventTypeGooseQuacked: event.NewTypedHandler[external.EventGooseQuacked](duckService.HandleGooseQuacked),
	})

	handlerHub := hub.Run(
		message.NewListener(
			duckEventMessageHandler, duckTopicConsumer,
			message.WithMetrics(metrics),
			message.WithLogging(logger, log.LevelInfo, log.LevelWarn),
		),
		message.NewListener(
			gooseEventMessageHandler, gooseTopicConsumer,
			message.WithMetrics(metrics),
			message.WithLogging(logger, log.LevelInfo, log.LevelWarn),
		),
	)

	logger.Info(ctx, "app is ready")
	hub.Must(handlerHub.Wait(ctx, sig.TermSignals(), logger))
}
