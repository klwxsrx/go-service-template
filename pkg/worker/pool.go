//go:generate mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Pool=Pool"
package worker

import (
	"errors"
	"runtime"
	"sync"
)

const WorkersCountNumCPU = 0 // TODO: unlimited workers count, optimal version for large workers count

var ErrPoolClosed = errors.New("pool is already closed")

type Job func()

type Pool interface {
	Do(Job) error
	Wait()
	Close()
}

type pool struct {
	wg      *sync.WaitGroup
	jobChan chan Job

	closeMutex *sync.RWMutex
	isClosed   bool
}

func (p *pool) Do(j Job) error {
	p.closeMutex.RLock()
	defer p.closeMutex.RUnlock()

	if p.isClosed {
		return ErrPoolClosed
	}

	p.wg.Add(1)
	p.jobChan <- j
	return nil
}

func (p *pool) Wait() {
	p.wg.Wait()
}

func (p *pool) Close() {
	p.closeMutex.Lock()
	defer p.closeMutex.Unlock()

	if p.isClosed {
		return
	}

	p.isClosed = true
	close(p.jobChan)
	p.Wait()
}

func runWorker(jobChan <-chan Job, jobDone func()) {
	go func() {
		for {
			job, ok := <-jobChan
			if !ok {
				break
			}

			job()
			jobDone()
		}
	}()
}

func NewPool(workersCount int) Pool {
	if workersCount <= WorkersCountNumCPU {
		workersCount = runtime.NumCPU()
	}

	jobChan := make(chan Job)
	wg := &sync.WaitGroup{}
	for i := 0; i < workersCount; i++ {
		runWorker(jobChan, wg.Done)
	}

	return &pool{
		wg:         wg,
		jobChan:    jobChan,
		closeMutex: &sync.RWMutex{},
		isClosed:   false,
	}
}
