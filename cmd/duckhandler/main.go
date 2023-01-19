package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/integration"
	duckappmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	duckintegrationmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/integration/message"
	"github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	app, ctx, logger := cmd.StartApp(log.LevelInfo)
	defer app.Finish(ctx)

	logger.Info(ctx, "app is starting")

	sqlConn := cmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlConn.Close(ctx)

	pulsarConn := cmd.MustInitPulsar(nil)
	defer pulsarConn.Close()

	container := pkgduck.NewDependencyContainer(ctx, sqlConn, pulsarConn, logger)
	defer container.Close()

	duckTopicConsumer := cmd.MustInitPulsarFailoverConsumer(pulsarConn, duckappmessage.DuckDomainEventTopicName, pkgduck.MessageSubscriberServiceName)
	defer duckTopicConsumer.Close()

	gooseTopicConsumer := cmd.MustInitPulsarFailoverConsumer(pulsarConn, duckintegrationmessage.GooseDomainEventTopicName, pkgduck.MessageSubscriberServiceName)
	defer gooseTopicConsumer.Close()

	duckService := container.DuckService()
	duckEventMessageHandler := message.NewEventHandler(duckappmessage.NewEventDeserializer(), message.EventTypeHandlerMap{
		domain.EventTypeDuckCreated: event.NewTypedHandler[domain.EventDuckCreated](duckService.HandleDuckCreated),
	})
	gooseEventMessageHandler := message.NewEventHandler(duckintegrationmessage.NewGooseEventDeserializer(), message.EventTypeHandlerMap{
		integration.EventTypeGooseQuacked: event.NewTypedHandler[integration.EventGooseQuacked](duckService.HandleGooseQuacked),
	})

	handlerHub := hub.Run(
		message.NewHandlerProcess(duckEventMessageHandler, duckTopicConsumer, logger),
		message.NewHandlerProcess(gooseEventMessageHandler, gooseTopicConsumer, logger),
	)

	logger.Info(ctx, "app is ready")
	hub.Must(handlerHub.Wait(sig.TermSignals()))
}
