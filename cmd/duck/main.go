package main

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	"github.com/klwxsrx/go-service-template/pkg/cmd"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	ctx := context.Background()
	logger := log.New(log.LevelInfo)
	defer cmd.HandleAppPanic(ctx, logger)

	logger.Info(ctx, "app is starting")

	sqlConn := cmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlConn.Close(ctx)

	pulsarConn := cmd.MustInitPulsar(nil)
	defer pulsarConn.Close()

	container := pkgduck.NewDependencyContainer(ctx, sqlConn, pulsarConn, logger)
	defer container.Close()

	httpServer := pkghttp.NewServer(
		pkghttp.DefaultServerAddress,
		pkghttp.WithHealthCheck(nil),
		pkghttp.WithRequestID(
			pkghttp.NewHTTPHeaderRequestIDExtractor(pkghttp.DefaultRequestIDHeader),
			pkghttp.NewRandomUUIDRequestIDExtractor(),
		),
		pkghttp.WithLogging(logger),
	)

	httpServer.Register(http.NewCreateDuckHandler(container.DuckService()))

	logger.Info(ctx, "app is ready")
	pkghttp.Must(httpServer.ListenAndServe(ctx, sig.TermSignals()))
}
