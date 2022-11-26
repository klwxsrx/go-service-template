package log

import (
	"context"
	"github.com/rs/zerolog"
	"os"
)

type Level int

const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Fields map[string]any

type Logger interface {
	With(fields Fields) Logger
	WithField(name string, value any) Logger
	WithError(err error) Logger
	Debug(ctx context.Context, str string)
	Debugf(ctx context.Context, format string, v ...any)
	Error(ctx context.Context, str string)
	Errorf(ctx context.Context, format string, v ...any)
	Warn(ctx context.Context, str string)
	Warnf(ctx context.Context, format string, v ...any)
	Info(ctx context.Context, str string)
	Infof(ctx context.Context, format string, v ...any)
	Fatal(ctx context.Context, str string)
	Fatalf(ctx context.Context, format string, v ...any)
}

type logger struct {
	impl *zerolog.Logger
}

func (l *logger) With(fields Fields) Logger {
	z := l.impl.With().Fields(fields).Logger()
	return &logger{&z}
}

func (l *logger) WithField(name string, v any) Logger {
	z := l.impl.With().Fields([]any{name, v}).Logger()
	return &logger{&z}
}

func (l *logger) WithError(err error) Logger {
	z := l.impl.With().Stack().Err(err).Logger()
	return &logger{&z}
}

func (l *logger) Debug(_ context.Context, str string) {
	l.impl.Debug().Msg(str)
}

func (l *logger) Debugf(_ context.Context, format string, v ...any) {
	l.impl.Debug().Msgf(format, v...)
}

func (l *logger) Error(_ context.Context, str string) {
	l.impl.Error().Msg(str)
}

func (l *logger) Errorf(_ context.Context, format string, v ...any) {
	l.impl.Error().Msgf(format, v...)
}

func (l *logger) Warn(_ context.Context, str string) {
	l.impl.Warn().Msg(str)
}

func (l *logger) Warnf(_ context.Context, format string, v ...any) {
	l.impl.Warn().Msgf(format, v...)
}

func (l *logger) Info(_ context.Context, str string) {
	l.impl.Info().Msg(str)
}

func (l *logger) Infof(_ context.Context, format string, v ...any) {
	l.impl.Info().Msgf(format, v...)
}

func (l *logger) Fatal(_ context.Context, str string) {
	l.impl.Fatal().Msg(str)
}

func (l *logger) Fatalf(_ context.Context, format string, v ...any) {
	l.impl.Fatal().Msgf(format, v...)
}

func New(lvl Level) Logger {
	var zl zerolog.Level
	switch lvl {
	case LevelDebug:
		zl = zerolog.DebugLevel
	case LevelInfo:
		zl = zerolog.InfoLevel
	case LevelWarn:
		zl = zerolog.WarnLevel
	case LevelError:
		zl = zerolog.ErrorLevel
	}

	z := zerolog.New(os.Stdout).Level(zl).With().Timestamp().Logger()
	return &logger{impl: &z}
}
