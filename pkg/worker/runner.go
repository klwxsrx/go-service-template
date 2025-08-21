package worker

import (
	"context"
	"time"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

func PeriodicRunner(job Job, every time.Duration) ErrorJob {
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

func WrapJobError(job ErrorJob, logger log.Logger) Job {
	return func(ctx context.Context) {
		err := job(ctx)
		if err != nil {
			logger.
				WithError(err).
				Error(ctx, "process completed with error")
		}
	}
}
