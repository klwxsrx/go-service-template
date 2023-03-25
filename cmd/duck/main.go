package main

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	pkgcmd "github.com/klwxsrx/go-service-template/pkg/cmd"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
	pkgobservability "github.com/klwxsrx/go-service-template/pkg/observability"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	ctx := context.Background()
	logger := log.New(log.LevelInfo)
	metrics := pkgmetricstub.NewMetrics()
	observability := pkgobservability.New()
	defer pkgcmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlConn := pkgcmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlConn.Close(ctx)

	pulsarConn := pkgcmd.MustInitPulsar(nil)
	defer pulsarConn.Close()

	gooseClient := cmd.MustInitGooseHTTPClient(observability, metrics, logger)

	container := pkgduck.NewDependencyContainer(ctx, sqlConn, pulsarConn, gooseClient, logger)
	defer container.Close()

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
		pkghttp.WithLogging(logger, log.LevelInfo, log.LevelWarn),
	)

	httpServer.Register(http.NewCreateDuckHandler(container.DuckService()))

	logger.Info(ctx, "app is ready")
	pkghttp.Must(httpServer.ListenAndServe(ctx, sig.TermSignals()))
}
