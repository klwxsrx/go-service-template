package main

import (
	"context"
	_ "github.com/joho/godotenv/autoload"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/sig"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func main() {
	ctx := context.Background()
	logger := log.New(log.LevelInfo)
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
