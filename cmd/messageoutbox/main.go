package main

import (
	"context"
	"time"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	logger := infra.Logger.MustLoad()

	pkgworker.MustRunHub(ctx, logger,
		pkgcmd.TermSignalAwaiter,
		pkgworker.PeriodicRunner(
			infra.MessageOutbox.MustLoad().Process,
			time.Second,
		),
	)
}
