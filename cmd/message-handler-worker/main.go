package main

import (
	"context"
	"time"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/userprofile"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	userProfile := userprofile.NewDependencyContainer(
		infra.DB,
		infra.DBMigrations,
		infra.HTTPClientFactory,
		infra.IdempotencyKeys,
	)

	messageBus := infra.MessageBusListener.MustLoad()
	userProfile.MustRegisterMessageHandlers(messageBus)

	messageStorageConsumers := infra.MessageStorageConsumers.MustLoad()
	messageHandlerWorkers := append(messageStorageConsumers.Workers(), messageBus.Workers()...)
	worker.MustRunHub(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		append(
			messageHandlerWorkers,
			worker.PeriodicRunner(func(context.Context) { messageStorageConsumers.Process() }, time.Second),
		)...,
	)
}
