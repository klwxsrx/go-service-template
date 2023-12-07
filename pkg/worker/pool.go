//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Pool=Pool"
package worker

import (
	"runtime"
	"sync"
)

const (
	MaxWorkersCountNumCPU    = -1
	MaxWorkersCountUnlimited = 0
)

type SimpleJob func()

type Pool interface {
	Do(SimpleJob)
	Wait()
}

type pool struct {
	jobCompleted    *sync.WaitGroup
	workerAvailable *sync.Cond
	currentWorkers  int
	maxWorkers      int
}

func NewPool(maxWorkers int) Pool {
	if maxWorkers <= MaxWorkersCountNumCPU {
		maxWorkers = runtime.NumCPU()
	}
	return &pool{
		jobCompleted:    &sync.WaitGroup{},
		workerAvailable: sync.NewCond(&sync.Mutex{}),
		currentWorkers:  0,
		maxWorkers:      maxWorkers,
	}
}

func (p *pool) Do(job SimpleJob) {
	p.jobCompleted.Add(1)

	if p.maxWorkers > 0 {
		p.workerAvailable.L.Lock()
		for p.currentWorkers >= p.maxWorkers {
			p.workerAvailable.Wait()
		}
		p.currentWorkers++
		p.workerAvailable.L.Unlock()
	}

	go func() {
		job()
		p.jobCompleted.Done()

		if p.maxWorkers > 0 {
			p.workerAvailable.L.Lock()
			p.currentWorkers--
			p.workerAvailable.L.Unlock()
			p.workerAvailable.Signal()
		}
	}()
}

func (p *pool) Wait() {
	p.jobCompleted.Wait()
}
