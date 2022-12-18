package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/event"
	duckappmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	app, ctx, logger := cmd.StartApp(log.LevelInfo)
	defer app.Finish(ctx)

	logger.Info(ctx, "app is starting")

	pulsarConn := cmd.MustInitPulsar(logger)
	defer pulsarConn.Close()

	duckTopicConsumer := cmd.MustInitPulsarSingleConsumer(pulsarConn, duckappmessage.DuckDomainEventTopicName, cmd.ServiceName)
	defer duckTopicConsumer.Close()

	duckCreatedEventHandler := event.NewDuckCreatedHandler()

	duckEventSerializer := duckappmessage.NewEventSerializer()
	eventMessageHandler := message.NewEventHandler(duckEventSerializer)
	eventMessageHandler.Subscribe(domain.EventTypeDuckCreated, duckCreatedEventHandler)

	handlerHub := hub.Run([]hub.Process{
		message.NewHandlerProcess(eventMessageHandler, duckTopicConsumer, logger),
	})

	logger.Info(ctx, "app is ready")
	hub.Must(handlerHub.Wait(sig.TermSignals()))
}
