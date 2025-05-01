package observability

import "context"

type observerStub struct{}

func NewStub() Observer {
	return observerStub{}
}

func (s observerStub) Field(_ context.Context, _ Field) string {
	return ""
}

func (s observerStub) WithField(ctx context.Context, _ Field, _ string) context.Context {
	return ctx
}
