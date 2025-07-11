package worker

import (
	"context"
	"sync"
)

type ErrorJob func(context.Context) error

type Group interface {
	Do(ErrorJob)
	Wait() error
}

type group struct {
	ctx                 context.Context
	ctxCancel           context.CancelFunc
	cancelCtxAfterError bool

	errChan   chan error
	errResult error
	pool      Pool
	wg        *sync.WaitGroup

	onceCloser *sync.Once
}

func WithinFailFastGroup(ctx context.Context, pool Pool) Group {
	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	return &group{
		ctx:                 ctx,
		ctxCancel:           ctxCancel,
		cancelCtxAfterError: true,
		errChan:             make(chan error, 1),
		errResult:           nil,
		pool:                pool,
		wg:                  &sync.WaitGroup{},
		onceCloser:          &sync.Once{},
	}
}

func WithinFailSafeGroup(ctx context.Context, pool Pool) Group {
	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	return &group{
		ctx:                 ctx,
		ctxCancel:           ctxCancel,
		cancelCtxAfterError: false,
		errChan:             make(chan error, 1),
		errResult:           nil,
		pool:                pool,
		wg:                  &sync.WaitGroup{},
		onceCloser:          &sync.Once{},
	}
}

func NewFailFastGroup(ctx context.Context) Group {
	return WithinFailFastGroup(
		ctx,
		NewPool(MaxWorkersCountUnlimited),
	)
}

func NewFailSafeGroup(ctx context.Context) Group {
	return WithinFailSafeGroup(
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
			if g.cancelCtxAfterError {
				g.ctxCancel()
			}
		default:
		}
	}

	g.wg.Add(1)
	g.pool.Do(g.ctx, func(ctx context.Context) {
		handleErr(job(ctx))
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
