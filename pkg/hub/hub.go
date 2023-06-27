package hub

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"os"
	"sync"
)

type Process interface {
	Name() string
	Func() func(stopChan <-chan struct{}) error
}

func Must(err error) {
	if err != nil {
		panic(fmt.Errorf("hub completed with error: %w", err))
	}
}

type Hub interface { // TODO: replace it by worker.Hub
	Wait(ctx context.Context, termSignalsChan <-chan os.Signal, logger log.Logger) error
}

type hub struct {
	wg                *sync.WaitGroup
	processCount      int
	onceDoer          *sync.Once
	stopChan          chan struct{}
	processResultChan chan processResult
	result            error
}

func (h *hub) Wait(ctx context.Context, termSignalsChan <-chan os.Signal, logger log.Logger) error {
	h.onceDoer.Do(func() {
		select {
		case <-termSignalsChan:
		case processResult := <-h.processResultChan:
			h.result = fmt.Errorf("process %s unexpectedly completed with error: %w", processResult.processName, processResult.err)
		}

		for i := 0; i < h.processCount; i++ {
			h.stopChan <- struct{}{}
		}

		h.wg.Wait()

		for {
			select {
			case processResult := <-h.processResultChan:
				if processResult.err != nil {
					logger.WithField("processName", processResult.processName).
						WithError(processResult.err).
						Error(ctx, "process completed after stop with error")
				}
			default:
				return
			}
		}
	})

	h.wg.Wait()
	return h.result
}

func Run(ps ...Process) Hub {
	wg := &sync.WaitGroup{}
	stopChan := make(chan struct{}, len(ps))
	processResultChan := make(chan processResult, len(ps))

	for _, p := range ps {
		wg.Add(1)
		go func(p Process) {
			err := p.Func()(stopChan)
			processResultChan <- processResult{p.Name(), err}
			wg.Done()
		}(p)
	}

	return &hub{
		wg:                wg,
		processCount:      len(ps),
		processResultChan: processResultChan,
		onceDoer:          &sync.Once{},
		stopChan:          stopChan,
	}
}

type processResult struct {
	processName string
	err         error
}
