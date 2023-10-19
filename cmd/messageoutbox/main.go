package main

import (
	"context"

	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgsig "github.com/klwxsrx/go-service-template/pkg/sig"
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	logger := pkgcmd.InitLogger()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, nil) // TODO: add outbox migrations
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	msgOutboxProcessor := pkgmessage.NewOutboxProcessor(
		pkgmessage.DefaultOutboxProcessingInterval,
		pkgcmd.MustInitSQLMessageOutbox(ctx, sqlDB, msgBroker, logger),
	)

	logger.Info(ctx, "app is ready")
	pkgworker.Must(pkgworker.Run(msgOutboxProcessor).Wait(ctx, pkgsig.TermSignals(), logger))
}
