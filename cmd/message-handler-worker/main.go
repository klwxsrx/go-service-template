package main

import (
	"time"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/userprofile"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx, ctxCancel := pkgcmd.InitializeContext()
	defer ctxCancel()

	infra := cmd.NewInfrastructure(ctx)
	defer infra.Close(ctx)

	userProfile := userprofile.NewContainer(
		infra.DB,
		infra.DBMigrations,
		infra.HTTPClientFactory,
		infra.IdempotencyKeys,
	)

	messageBus := infra.MessageBusListener.MustLoad()
	userProfile.Services.MustRegister(messageBus)

	messageStorageConsumers := infra.MessageStorageConsumers.MustLoad()
	messageHandlerWorkers := append(messageStorageConsumers.Workers(), messageBus.Workers()...)
	pkgcmd.MustRun(ctx, infra.Logger.MustLoad(), append(
		messageHandlerWorkers,
		worker.PeriodicalJob(messageStorageConsumers.Process, time.Second),
	)...)
}
