//go:generate mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Logger=Logger"
package log

import (
	"context"
	"github.com/rs/zerolog"
	"os"
)

type contextKey int

const (
	fieldsContextKey contextKey = iota
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
	WithContext(ctx context.Context, fields Fields) context.Context
	Debug(ctx context.Context, str string)
	Error(ctx context.Context, str string)
	Warn(ctx context.Context, str string)
	Info(ctx context.Context, str string)
	Fatal(ctx context.Context, str string)
}

type logger struct {
	impl zerolog.Logger
}

func (l logger) With(fields Fields) Logger {
	z := l.impl.With().Fields(map[string]any(fields)).Logger()
	return logger{z}
}

func (l logger) WithField(name string, v any) Logger {
	z := l.impl.With().Fields([]any{name, v}).Logger()
	return logger{z}
}

func (l logger) WithError(err error) Logger {
	z := l.impl.With().Stack().Err(err).Logger()
	return logger{z}
}

func (l logger) WithContext(ctx context.Context, fields Fields) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	ctxFields := l.getFieldsFromContext(ctx)
	for key, value := range fields {
		ctxFields[key] = value
	}
	return context.WithValue(ctx, fieldsContextKey, ctxFields)
}

func (l logger) Debug(ctx context.Context, str string) {
	l.loggerWithContextFields(ctx).Debug().Msg(str)
}

func (l logger) Error(ctx context.Context, str string) {
	l.loggerWithContextFields(ctx).Error().Msg(str)
}

func (l logger) Warn(ctx context.Context, str string) {
	l.loggerWithContextFields(ctx).Warn().Msg(str)
}

func (l logger) Info(ctx context.Context, str string) {
	l.loggerWithContextFields(ctx).Info().Msg(str)
}

func (l logger) Fatal(ctx context.Context, str string) {
	l.loggerWithContextFields(ctx).Fatal().Msg(str)
}

func (l logger) loggerWithContextFields(ctx context.Context) *zerolog.Logger {
	z := l.impl.With().Fields(map[string]any(l.getFieldsFromContext(ctx))).Logger()
	return &z
}

func (l logger) getFieldsFromContext(ctx context.Context) Fields {
	fields, ok := ctx.Value(fieldsContextKey).(Fields)
	if !ok {
		return make(Fields)
	}
	return fields
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
	return logger{impl: z}
}
