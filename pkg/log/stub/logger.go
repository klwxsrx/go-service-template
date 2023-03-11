package stub

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/log"
)

type logger struct{}

func (l logger) With(_ log.Fields) log.Logger {
	return l
}

func (l logger) WithField(_ string, _ any) log.Logger {
	return l
}

func (l logger) WithError(_ error) log.Logger {
	return l
}

func (l logger) WithContext(ctx context.Context, _ log.Fields) context.Context {
	return ctx
}

func (l logger) Debug(_ context.Context, _ string) {}

func (l logger) Error(_ context.Context, _ string) {}

func (l logger) Warn(_ context.Context, _ string) {}

func (l logger) Info(_ context.Context, _ string) {}

func (l logger) Fatal(_ context.Context, _ string) {}

func NewLogger() log.Logger {
	return &logger{}
}
