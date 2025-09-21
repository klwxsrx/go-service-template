package worker

import (
	"context"
	"time"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

type ContextJob func(context.Context) error

func PeriodicalJob(job Job, every time.Duration) ContextJob {
	return periodicalImpl(func(context.Context) { job() }, every)
}

func PeriodicalContextJob(job ContextJob, every time.Duration, logger log.Logger) ContextJob {
	return periodicalImpl(func(ctx context.Context) {
		if err := job(ctx); err != nil {
			logger.WithError(err).Error(ctx, "periodical job completed with error")
		}
	}, every)
}

func periodicalImpl(job func(context.Context), every time.Duration) ContextJob {
	return func(ctx context.Context) error {
		ticker := time.NewTicker(every)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				job(ctx)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
