package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	pkgduck "github.com/klwxsrx/go-service-template/internal/pkg/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	app, ctx, logger := cmd.StartApp(log.LevelInfo)
	defer app.Finish(ctx)

	logger.Info(ctx, "app is starting")

	sqlConn := cmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlConn.Close(ctx)

	pulsarConn := cmd.MustInitPulsar(logger)
	defer pulsarConn.Close()

	container := pkgduck.NewDependencyContainer(ctx, sqlConn, pulsarConn, logger)
	defer container.Close()

	httpServer := pkghttp.NewServer(
		pkghttp.DefaultServerAddress,
		logger,
		pkghttp.WithHealthCheck(nil),
		pkghttp.WithLogging(logger),
	)
	defer httpServer.Shutdown(ctx)

	httpServer.Register(http.NewCreateDuckHandler(container.DuckService()))
	httpServer.MustListenAndServe()

	logger.Info(ctx, "app is ready")
	<-sig.TermSignals()
}
