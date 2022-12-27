package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/integration"
	duckappmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	duckintegrationmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/integration/message"
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

	pulsarConn := cmd.MustInitPulsar(logger)
	defer pulsarConn.Close()

	container := pkgduck.NewDependencyContainer(ctx, sqlConn, pulsarConn, logger)
	defer container.Close()

	duckTopicConsumer := cmd.MustInitPulsarSingleConsumer(pulsarConn, duckappmessage.DuckDomainEventTopicName, cmd.DuckServiceName)
	defer duckTopicConsumer.Close()

	gooseTopicConsumer := cmd.MustInitPulsarSingleConsumer(pulsarConn, duckintegrationmessage.GooseDomainEventTopicName, cmd.DuckServiceName)
	defer gooseTopicConsumer.Close()

	duckEventMessageHandler := message.NewEventHandlerComposite(duckappmessage.NewEventTypeDecoder())
	duckEventMessageHandler.Subscribe(message.EventTypeHandlerMap{
		domain.EventTypeDuckCreated: message.NewEventHandler[domain.EventDuckCreated](
			duckappmessage.EventSerializerDuckCreated,
			container.DuckService().HandleDuckCreated,
		),
	})

	gooseEventMessageHandler := message.NewEventHandlerComposite(duckintegrationmessage.NewEventTypeDecoder())
	gooseEventMessageHandler.Subscribe(message.EventTypeHandlerMap{
		integration.EventTypeGooseQuacked: message.NewEventHandler[integration.EventGooseQuacked](
			duckintegrationmessage.EventSerializerGooseQuacked,
			container.DuckService().HandleGooseQuacked,
		),
	})

	handlerHub := hub.Run([]hub.Process{
		message.NewHandlerProcess(duckEventMessageHandler, duckTopicConsumer, logger),
		message.NewHandlerProcess(gooseEventMessageHandler, gooseTopicConsumer, logger),
	})

	logger.Info(ctx, "app is ready")
	hub.Must(handlerHub.Wait(sig.TermSignals()))
}
