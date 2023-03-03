package log

import "context"

type stub struct{}

func (s *stub) With(_ Fields) Logger {
	return s
}

func (s *stub) WithField(_ string, _ any) Logger {
	return s
}

func (s *stub) WithError(_ error) Logger {
	return s
}

func (s *stub) WithContext(ctx context.Context, _ Fields) context.Context {
	return ctx
}

func (s *stub) Debug(_ context.Context, _ string) {}

func (s *stub) Error(_ context.Context, _ string) {}

func (s *stub) Warn(_ context.Context, _ string) {}

func (s *stub) Info(_ context.Context, _ string) {}

func (s *stub) Fatal(_ context.Context, _ string) {}

func NewStub() Logger {
	return &stub{}
}
