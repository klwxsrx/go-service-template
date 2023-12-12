package main

import (
	"context"

	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgsig "github.com/klwxsrx/go-service-template/pkg/sig"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
	pkgworker "github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	logger := pkgcmd.InitLogger()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, pkgsql.MessageOutboxMigrations)
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	msgOutbox := pkgcmd.MustInitSQLMessageOutbox(sqlDB, msgBroker, logger)
	defer msgOutbox.Close()

	logger.Info(ctx, "app is ready")
	pkgworker.MustRunHub(ctx, logger,
		pkgsig.TermSignalAwaiter,
		pkgmessage.NewOutboxProcessor(msgOutbox, pkgmessage.DefaultOutboxProcessingInterval),
	)
}
