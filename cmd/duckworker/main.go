package main

import (
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/cmd"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/sig"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	app, ctx, logger := cmd.StartApp(log.LevelInfo)
	defer app.Finish(ctx)

	logger.Info(ctx, "app is starting")

	pool := worker.NewPool(worker.NumCPUWorkersCount)
	defer pool.Close()

	logger.Info(ctx, "app is ready")

	_ = pool.Do(func() {
		logger.Info(ctx, "job 1 done")
	})
	_ = pool.Do(func() {
		logger.Info(ctx, "job 2 done")
	})
	_ = pool.Do(func() {
		logger.Info(ctx, "job 3 done")
	})

	<-sig.TermSignals()
}
