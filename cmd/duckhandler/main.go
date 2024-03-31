package main

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	container := duck.NewDependencyContainer(
		infra.DB,
		infra.DBMigrations,
		infra.EventDispatcher,
		infra.HTTPClientFactory,
	)

	msgBusListener := infra.MessageBusListener.MustLoad()
	container.MustRegisterMessageHandlers(msgBusListener)

	pkgworker.MustRunHub(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		msgBusListener.Workers()...,
	)
}
