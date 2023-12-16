package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

func MustRunHub(ctx context.Context, logger log.Logger, process Process, processes ...Process) {
	err := RunHub(ctx, logger, process, processes...)
	if err != nil {
		panic(fmt.Errorf("process completed with error: %w", err))
	}
}

func RunHub(ctx context.Context, logger log.Logger, process Process, processes ...Process) error {
	errProcessCompleted := errors.New("process completed")
	loggingWrapper := func(process Process, logger log.Logger) Process {
		return func(ctx context.Context) error {
			err := process(ctx)
			if errors.Is(err, context.Canceled) {
				return err
			}
			if err == nil {
				return errProcessCompleted
			}

			logger.WithError(err).Error(ctx, "running process completed with error")
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
