package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

func MustRunHub(ctx context.Context, logger log.Logger, process ErrorJob, processes ...ErrorJob) {
	err := RunHub(ctx, logger, process, processes...)
	if err != nil {
		panic(fmt.Errorf("process completed with error: %w", err))
	}
}

func RunHub(ctx context.Context, logger log.Logger, process ErrorJob, processes ...ErrorJob) error {
	errProcessCompleted := errors.New("process completed")
	loggingWrapper := func(process ErrorJob, logger log.Logger) ErrorJob {
		return func(ctx context.Context) error {
			err := process(ctx)
			if errors.Is(err, context.Canceled) { // TODO: check errors.Is(ctx.Err(), context.Canceled) to split context cancellation and business logic
				return err
			}
			if err == nil {
				return errProcessCompleted
			}

			logger.WithError(err).Error(ctx, "process completed with error")
			return err
		}
	}

	processGroup := NewFailFastGroup(ctx)
	processGroup.Do(loggingWrapper(process, logger))
	for _, process := range processes {
		processGroup.Do(loggingWrapper(process, logger))
	}

	err := processGroup.Wait()
	if errors.Is(err, errProcessCompleted) || errors.Is(err, context.Canceled) {
		return nil
	}

	return err
}
