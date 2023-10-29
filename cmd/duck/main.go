package main

import (
	"context"

	sqlduck "github.com/klwxsrx/go-service-template/data/sql/duck"
	"github.com/klwxsrx/go-service-template/internal/duck"
	commonhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	pkglog "github.com/klwxsrx/go-service-template/pkg/log"
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
	pkgobservability "github.com/klwxsrx/go-service-template/pkg/observability"
	pkgsig "github.com/klwxsrx/go-service-template/pkg/sig"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

func main() {
	ctx := context.Background()
	logger := pkgcmd.InitLogger()
	metrics := pkgmetricstub.NewMetrics()
	observability := pkgobservability.New()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlDB := pkgcmd.MustInitSQL(ctx, logger, pkgsql.MessageOutboxMigrations, sqlduck.Migrations)
	defer sqlDB.Close(ctx)

	msgBroker := pkgcmd.MustInitPulsarMessageBroker(nil)
	defer msgBroker.Close()

	msgOutbox := pkgcmd.MustInitSQLMessageOutbox(sqlDB, msgBroker, logger)
	defer msgOutbox.Close()

	httpClients := commonhttp.NewClientFactory(observability, metrics, logger)

	container := duck.MustInitDependencyContainer(sqlDB, httpClients, msgOutbox.Process)

	httpServer := pkghttp.NewServer(
		pkghttp.DefaultServerAddress,
		pkghttp.WithHealthCheck(nil),
		pkghttp.WithCORSHandler(),
		pkghttp.WithObservability(
			observability,
			pkghttp.NewHTTPHeaderRequestIDExtractor(pkghttp.DefaultRequestIDHeader),
			pkghttp.NewRandomUUIDRequestIDExtractor(),
		),
		pkghttp.WithMetrics(metrics),
		pkghttp.WithLogging(logger, pkglog.LevelInfo, pkglog.LevelError),
	)
	container.MustRegisterHTTPHandlers(httpServer)

	logger.Info(ctx, "app is ready")
	pkghttp.Must(httpServer.Listen(ctx, pkgsig.TermSignals()))
}
