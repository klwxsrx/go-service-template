package main

import (
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/user"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
)

func main() {
	ctx, ctxCancel := pkgcmd.InitializeContext()
	defer ctxCancel()

	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	container := user.NewDependencyContainer(
		infra.DB,
		infra.DBMigrations,
		infra.EventDispatcher,
	)

	httpServer := infra.HTTPServer.MustLoad()
	container.MustRegisterHTTPHandlers(httpServer)

	pkgcmd.MustRun(ctx, infra.Logger.MustLoad(),
		httpServer.Listener,
	)
}
