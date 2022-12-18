package hub

import (
	"fmt"
	"os"
	"sync"
)

type Process func(stopChan <-chan struct{})

func Must(err error) {
	if err != nil {
		panic(fmt.Errorf("hub completed with error: %w", err))
	}
}

type Hub interface {
	Wait(termSignalsChan <-chan os.Signal) error
}

type hub struct {
	wg              *sync.WaitGroup
	processCount    int
	processDoneChan chan int
	onceDoer        *sync.Once
	stopChan        chan struct{}
	result          error
}

func (h *hub) Wait(termSignalsChan <-chan os.Signal) error {
	h.onceDoer.Do(func() {
		select {
		case <-termSignalsChan:
		case position := <-h.processDoneChan:
			h.result = fmt.Errorf("process %d unexpectedly completed", position)
		}

		for i := 0; i < h.processCount; i++ {
			h.stopChan <- struct{}{}
		}
	})

	h.wg.Wait()
	return h.result
}

func Run(ps []Process) Hub {
	wg := &sync.WaitGroup{}
	stopChan := make(chan struct{}, len(ps))
	processDoneChan := make(chan int, 1)

	for i, p := range ps {
		wg.Add(1)
		go func(p Process, position int) {
			p(stopChan)
			select {
			case processDoneChan <- position:
			default:
			}
			wg.Done()
		}(p, i)
	}

	return &hub{
		wg:              wg,
		processCount:    len(ps),
		processDoneChan: processDoneChan,
		onceDoer:        &sync.Once{},
		stopChan:        stopChan,
	}
}
