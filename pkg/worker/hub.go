package worker

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"os"
	"sync"
)

func Must(err error) {
	if err != nil {
		panic(fmt.Errorf("worker completed with error: %w", err))
	}
}

type (
	Process func(stopChan <-chan struct{}) error

	NamedProcess interface {
		Name() string
		Process() Process
	}

	Hub interface {
		Wait(context.Context, <-chan os.Signal, log.Logger) error
	}
)

type hub struct {
	pool       Pool
	stopChan   chan<- struct{}
	errChan    <-chan errProcessFailed
	onceCloser *sync.Once
	result     error
}

func (h *hub) Wait(ctx context.Context, termSignalChan <-chan os.Signal, logger log.Logger) error {
	h.onceCloser.Do(func() {
		select {
		case <-termSignalChan:
		case err := <-h.errChan:
			logger.WithField("processName", err.name).
				WithError(err.err).
				Error(ctx, "process unexpectedly completed with error")
			h.result = fmt.Errorf("process %s unexpectedly completed with error: %w", err.name, err.err)
		}

	stop:
		for {
			select {
			case h.stopChan <- struct{}{}:
			default:
				break stop
			}
		}

		for {
			select {
			case err := <-h.errChan:
				logger.WithField("processName", err.name).
					WithError(err.err).
					Error(ctx, "process completed after stop with error")
			default:
				return
			}
		}
	})

	h.pool.Wait()
	return h.result
}

func RunHub(ps ...NamedProcess) Hub {
	stopChan := make(chan struct{})
	errChan := make(chan errProcessFailed)

	pool := NewPool(MaxWorkersCountUnlimited)
	for _, p := range ps {
		proc := p
		pool.Do(func() {
			err := proc.Process()(stopChan)
			if err != nil {
				errChan <- errProcessFailed{
					name: proc.Name(),
					err:  err,
				}
			}
		})
	}

	return &hub{
		pool:       pool,
		stopChan:   stopChan,
		errChan:    errChan,
		onceCloser: &sync.Once{},
		result:     nil,
	}
}

type errProcessFailed struct {
	name string
	err  error
}
