package worker

import (
	"context"
	"sync"
)

type (
	ErrorJob func() error

	Group interface {
		Do(ErrorJob)
		Wait() error
	}
)

type group struct {
	pool       Pool
	wg         *sync.WaitGroup
	ctxCancel  context.CancelFunc
	onceCloser *sync.Once
	errChan    chan error
	errResult  error
}

// WithinGroup creates a group with context, which is canceled when one of the jobs fails.
// Group will use a specified Pool to process the jobs.
// Pass groupCtx to the jobs to process them fail-fast.
// Or use the parent context to process the jobs fail-safe.
func WithinGroup(ctx context.Context, pool Pool) (groupCtx context.Context, _ Group) {
	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	return ctx, &group{
		pool:       pool,
		wg:         &sync.WaitGroup{},
		ctxCancel:  ctxCancel,
		onceCloser: &sync.Once{},
		errChan:    make(chan error, 1),
		errResult:  nil,
	}
}

// NewGroup creates a group with context, which is canceled when one of the jobs fails.
// Pass groupCtx to the jobs to process them fail-fast.
// Or use the parent context to process the jobs fail-safe.
func NewGroup(ctx context.Context) (groupCtx context.Context, _ Group) {
	return WithinGroup(
		ctx,
		NewPool(MaxWorkersCountUnlimited),
	)
}

func (g *group) Do(job ErrorJob) {
	handleErr := func(err error) {
		if err == nil {
			return
		}

		select {
		case g.errChan <- err:
			g.ctxCancel()
		default:
		}
	}

	g.wg.Add(1)
	g.pool.Do(func() {
		handleErr(job())
		g.wg.Done()
	})
}

func (g *group) Wait() error {
	g.wg.Wait()
	g.onceCloser.Do(func() {
		g.ctxCancel()

		select {
		case g.errResult = <-g.errChan:
		default:
		}
	})

	return g.errResult
}
