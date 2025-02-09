package observability

import (
	"context"
)

type observerStub struct{}

func NewObserverStub() Observer {
	return observerStub{}
}

func (s observerStub) RequestID(_ context.Context) (string, bool) {
	return "", false
}

func (s observerStub) WithRequestID(ctx context.Context, _ string) context.Context {
	return ctx
}
