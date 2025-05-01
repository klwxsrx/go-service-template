package time

import (
	"context"
	"time"
)

const nowContextKey contextKey = iota

type (
	Clock interface {
		Now(context.Context) time.Time
	}

	AdjustableClock interface {
		Clock
		Set(context.Context, time.Time) context.Context
		Freeze(context.Context) context.Context
	}

	clockImpl  struct{}
	contextKey int
)

func NewAdjustableClock() AdjustableClock {
	return clockImpl{}
}

func (c clockImpl) Now(ctx context.Context) time.Time {
	if t, ok := ctx.Value(nowContextKey).(time.Time); ok {
		return t
	}

	return time.Now()
}

func (c clockImpl) Set(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, nowContextKey, t)
}

func (c clockImpl) Freeze(ctx context.Context) context.Context {
	if _, ok := ctx.Value(nowContextKey).(time.Time); ok {
		return ctx
	}

	return c.Set(ctx, time.Now())
}
