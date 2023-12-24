package main

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	infra := cmd.NewInfrastructureContainer(ctx)

	pkgworker.MustRunHub(ctx, infra.Logger.MustLoad(),
		pkgcmd.TermSignalAwaiter,
		pkgmessage.NewOutboxProcessor(infra.MessageOutbox.MustLoad(), pkgmessage.DefaultOutboxProcessingInterval),
	)
}
