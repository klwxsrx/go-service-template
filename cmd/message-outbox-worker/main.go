package main

import (
	"context"
	"time"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)
	defer infra.Close(ctx)

	msgOutbox := infra.MessageOutbox.MustLoad()

	worker.MustRunHub(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		msgOutbox.Worker,
		worker.PeriodicRunner(func(context.Context) { msgOutbox.Process() }, time.Second),
	)
}
