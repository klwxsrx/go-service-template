package stub

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

type observer struct{}

func (o observer) RequestID(_ context.Context) (string, bool) {
	return "", false
}

func (o observer) WithRequestID(ctx context.Context, _ string) context.Context {
	return ctx
}

func New() observability.Observer {
	return observer{}
}
