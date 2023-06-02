package worker

import (
	"context"
	"errors"
)

type GroupJob func(context.Context) error

type FailSafeGroupContext interface {
	Do(GroupJob)
	Wait() error
}

type FailFastGroupContext interface {
	Do(GroupJob)
	Close() error
}

type group struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	errChan   chan error
	pool      Pool
}

func (g group) Do(job GroupJob) {
	handleErr := func(err error) {
		if err == nil {
			return
		}

		select {
		case g.errChan <- err:
			if g.ctxCancel != nil {
				g.ctxCancel()
			}
		default:
		}
	}

	if g.ctx.Err() != nil {
		handleErr(errors.New("group context is already closed"))
	}
	err := g.pool.Do(func() {
		handleErr(job(g.ctx))
	})
	handleErr(err)
}

func (g group) Wait() error {
	g.pool.Wait()

	select {
	case err := <-g.errChan:
		return err
	default:
		return nil
	}
}

func (g group) Close() error {
	return g.Wait()
}

func WithFailFastContext(ctx context.Context, pool Pool) FailFastGroupContext {
	var ctxCancel context.CancelFunc
	ctx, ctxCancel = context.WithCancel(ctx)
	return group{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		errChan:   make(chan error, 1),
		pool:      pool,
	}
}

func WithFailSafeContext(ctx context.Context, pool Pool) FailSafeGroupContext {
	return group{
		ctx:       ctx,
		ctxCancel: nil,
		errChan:   make(chan error, 1),
		pool:      pool,
	}
}
