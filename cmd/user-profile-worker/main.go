package main

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/userprofile"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	container := userprofile.NewDependencyContainer(
		infra.DB,
		infra.DBMigrations,
		infra.HTTPClientFactory,
		infra.IdempotencyKeys,
	)

	messageBus := infra.MessageBusListener.MustLoad()
	container.MustRegisterMessageHandlers(messageBus)

	worker.MustRunHub(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		messageBus.Workers()...,
	)
}
