package main

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	worker.MustRunHub(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		infra.IdempotencyKeysCleaner.MustLoad().DeleteOutdated,
	)
}
