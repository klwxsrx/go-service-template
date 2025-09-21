package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

func MustRun(ctx context.Context, logger log.Logger, job ...worker.ContextJob) {
	if err := Run(ctx, logger, job...); err != nil {
		panic(fmt.Errorf("some of the jobs completed with error: %w", err))
	}
}

func Run(ctx context.Context, logger log.Logger, job ...worker.ContextJob) error {
	errCompleted := errors.New("job completed")
	loggingAdapter := func(ctx context.Context, job worker.ContextJob, logger log.Logger) worker.ErrorJob {
		return func() error {
			err := job(ctx)
			if err == nil || errors.Is(err, ctx.Err()) {
				return errCompleted
			}

			logger.WithError(err).Error(ctx, "running job completed with error")
			return err
		}
	}

	groupCtx, group := worker.NewGroup(ctx)
	for _, j := range job {
		group.Do(loggingAdapter(groupCtx, j, logger))
	}

	err := group.Wait()
	if !errors.Is(err, errCompleted) {
		return err
	}

	return nil
}
