package main

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	pkgcmd.MustRun(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		infra.IdempotencyKeysCleaner.MustLoad().DeleteOutdated,
	)
}
