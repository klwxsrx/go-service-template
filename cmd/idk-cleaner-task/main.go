package main

import (
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
)

func main() {
	ctx, ctxCancel := pkgcmd.InitializeContext()
	defer ctxCancel()

	infra := cmd.NewInfrastructure(ctx)
	defer infra.Close(ctx)

	idkCleaner := infra.IdempotencyKeysCleaner.MustLoad()
	pkgcmd.MustRun(ctx, infra.Logger.MustLoad(),
		idkCleaner.DeleteOutdated,
	)
}
