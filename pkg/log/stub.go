package log

import "context"

type stub struct {
}

func (s *stub) With(_ Fields) Logger {
	return s
}

func (s *stub) WithField(_ string, _ any) Logger {
	return s
}

func (s *stub) WithError(_ error) Logger {
	return s
}

func (s *stub) Debug(_ context.Context, _ string) {}

func (s *stub) Debugf(_ context.Context, _ string, _ ...any) {}

func (s *stub) Error(_ context.Context, _ string) {}

func (s *stub) Errorf(_ context.Context, _ string, _ ...any) {}

func (s *stub) Warn(_ context.Context, _ string) {}

func (s *stub) Warnf(_ context.Context, _ string, _ ...any) {}

func (s *stub) Info(_ context.Context, _ string) {}

func (s *stub) Infof(_ context.Context, _ string, _ ...any) {}

func (s *stub) Fatal(_ context.Context, _ string) {}

func (s *stub) Fatalf(_ context.Context, _ string, _ ...any) {}

func NewStub() Logger {
	return &stub{}
}
