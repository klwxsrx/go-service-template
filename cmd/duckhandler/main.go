package main

import (
	"context"
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
	ctx := context.Background()
	logger := log.New(log.LevelInfo)

	pulsarConn := cmd.MustInitPulsar(ctx, logger)
	defer pulsarConn.Close()

	duckTopicConsumer := cmd.MustInitPulsarSingleConsumer(ctx, pulsarConn, duckappmessage.DuckDomainEventTopicName, cmd.ServiceName, logger)
	defer duckTopicConsumer.Close()

	duckCreatedEventHandler := event.NewDuckCreatedHandler()

	duckEventSerializer := duckappmessage.NewEventSerializer()
	eventMessageHandler := message.NewEventHandler(duckEventSerializer)
	eventMessageHandler.Subscribe(domain.EventTypeDuckCreated, duckCreatedEventHandler)

	handlerHub := hub.Run([]hub.Process{
		message.NewHandlerProcess(eventMessageHandler, duckTopicConsumer, logger),
	})

	logger.Info(ctx, "app is ready")
	err := handlerHub.Wait(sig.TermSignals())
	if err != nil {
		logger.Fatalf(ctx, "handler hub completed with error: %w", err)
	}
}
