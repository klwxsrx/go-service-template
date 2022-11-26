package main

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/persistence"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/sig"
)

func main() {
	ctx := context.Background()
	logger := log.New(log.LevelInfo)

	sqlConn := cmd.MustInitSQL(ctx, logger, duck.SQLMigrations)
	defer sqlConn.Close(ctx)

	pulsarConn := cmd.MustInitPulsar(ctx, logger)
	defer pulsarConn.Close()

	messageOutbox := cmd.MustInitSQLMessageOutbox(sqlConn, pulsarConn, logger)
	defer messageOutbox.Close()

	ufw := persistence.NewUnitOfWork(sqlConn.Client(), message.NewEventSerializer(), func() {
		messageOutbox.Process()
	})
	duckService := service.NewDuckService(ufw)

	httpServer := pkghttp.NewServer(
		pkghttp.DefaultServerAddress,
		logger,
		pkghttp.WithHealthCheck(nil),
		pkghttp.WithLogging(logger),
	)
	defer httpServer.Shutdown(ctx)

	httpServer.Register(http.NewCreateDuckHandler(duckService))
	httpServer.MustListenAndServe(ctx)

	logger.Info(ctx, "app is ready")
	<-sig.TermSignals()
}
