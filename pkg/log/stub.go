package log

import (
	"context"
)

type stub struct{}

func (s stub) With(Fields) Logger {
	return s
}

func (s stub) WithField(string, any) Logger {
	return s
}

func (s stub) WithError(error) Logger {
	return s
}

func (s stub) WithContext(ctx context.Context, _ Fields) context.Context {
	return ctx
}

func (s stub) Debug(context.Context, string) {}

func (s stub) Error(context.Context, string) {}

func (s stub) Warn(context.Context, string) {}

func (s stub) Info(context.Context, string) {}

func (s stub) Fatal(context.Context, string) {}

func (s stub) Log(context.Context, Level, string) {}
