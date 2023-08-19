//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Group=Group"
package worker

import (
	"context"
	"sync"
)

type Job func(context.Context) error

type Group interface {
	Do(Job)
	Close() error
}

type group struct {
	ctx                 context.Context
	ctxCancel           context.CancelFunc
	cancelCtxAfterError bool

	errChan   chan error
	errResult error
	pool      Pool

	closerMutex *sync.Mutex
	onceCloser  *sync.Once
}

func (g *group) Do(job Job) {
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

	g.pool.Do(func() {
		handleErr(job(g.ctx))
	})
}

func (g *group) Close() error {
	g.pool.Wait()

	g.closerMutex.Lock()
	g.onceCloser.Do(func() {
		g.ctxCancel()

		select {
		case g.errResult = <-g.errChan:
		default:
		}
	})
	g.closerMutex.Unlock()

	return g.errResult
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
		closerMutex:         &sync.Mutex{},
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
		closerMutex:         &sync.Mutex{},
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
