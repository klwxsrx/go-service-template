package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/data/sql/duck"
	duckappmessage "github.com/klwxsrx/go-service-template/internal/pkg/duck/app/message"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/http"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/sql"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
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

	messageOutbox := cmd.MustInitSQLMessageOutbox(ctx, sqlConn, pulsarConn, logger)
	defer messageOutbox.Close()

	sqlClient, transaction := cmd.MustInitSQLTransaction(sqlConn, func() {
		messageOutbox.Process()
	})

	messageStore := cmd.MustInitSQLMessageStore(ctx, sqlClient)

	duckEventDispatcher := message.NewEventDispatcher(
		message.NewStoreSender(messageStore),
		duckappmessage.NewEventSerializer(),
	)

	duckRepo := sql.NewDuckRepo(sqlClient, duckEventDispatcher)
	duckService := service.NewDuckService(duckRepo, transaction)

	httpServer := pkghttp.NewServer(
		pkghttp.DefaultServerAddress,
		logger,
		pkghttp.WithHealthCheck(nil),
		pkghttp.WithLogging(logger),
	)
	defer httpServer.Shutdown(ctx)

	httpServer.Register(http.NewCreateDuckHandler(duckService))
	httpServer.MustListenAndServe()

	logger.Info(ctx, "app is ready")
	<-sig.TermSignals()
}
