//go:generate mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Pool=Pool"
package worker

import (
	"errors"
	"runtime"
	"sync"
)

const NumCPUWorkersCount = 0

var ErrPoolClosed = errors.New("pool is already closed")

type Job func()

type Pool interface {
	Do(j Job) error
	Close()
}

type pool struct {
	jobChan  chan Job
	wg       *sync.WaitGroup
	mutex    *sync.Mutex
	isClosed bool
}

func (p *pool) Do(j Job) (result error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isClosed {
		return ErrPoolClosed
	}

	p.jobChan <- j
	return
}

func (p *pool) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isClosed {
		return
	}

	p.isClosed = true
	close(p.jobChan)
	p.wg.Wait()
}

func runWorker(jobChan <-chan Job, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		for {
			job, ok := <-jobChan
			if !ok {
				break
			}
			job()
		}
		wg.Done()
	}()
}

func NewPool(workersCount int) Pool {
	jobChan := make(chan Job)
	wg := &sync.WaitGroup{}
	if workersCount <= NumCPUWorkersCount {
		workersCount = runtime.NumCPU()
	}

	for i := 0; i < workersCount; i++ {
		runWorker(jobChan, wg)
	}

	return &pool{
		jobChan: jobChan,
		wg:      wg,
		mutex:   &sync.Mutex{},
	}
}
