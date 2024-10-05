//go:generate ${TOOLS_BIN}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Logger=Logger"
package log

import (
	"context"
	"log/slog"
	"os"
)

const (
	LevelDisabled Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

type (
	Logger interface {
		With(fields Fields) Logger
		WithField(name string, value any) Logger
		WithError(err error) Logger
		WithContext(ctx context.Context, fields Fields) context.Context
		Log(ctx context.Context, lvl Level, msg string)
		Debug(ctx context.Context, msg string)
		Info(ctx context.Context, msg string)
		Warn(ctx context.Context, msg string)
		Error(ctx context.Context, msg string)
	}

	Fields map[string]any
	Level  int

	contextKey int
)

const fieldsContextKey contextKey = iota

var slogLevelMap = map[Level]slog.Level{
	LevelDebug: slog.LevelDebug,
	LevelInfo:  slog.LevelInfo,
	LevelWarn:  slog.LevelWarn,
	LevelError: slog.LevelError,
}

type logger struct {
	impl *slog.Logger
}

func New(level Level) Logger {
	if level == LevelDisabled {
		return stub{}
	}

	impl := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slogLevelMap[level]},
	))

	return logger{impl}
}

func (l logger) With(fields Fields) Logger {
	if len(fields) == 0 {
		return l
	}

	l.impl = l.impl.With(convertFields(fields)...)
	return l
}

func (l logger) WithField(name string, v any) Logger {
	l.impl = l.impl.With(name, v)
	return l
}

func (l logger) WithError(err error) Logger {
	if err == nil {
		return l
	}

	l.impl = l.impl.With("error", err.Error())
	return l
}

func (l logger) WithContext(ctx context.Context, fields Fields) context.Context {
	if len(fields) == 0 {
		return ctx
	}

	ctxFields := getContextFields(ctx)
	result := make([]any, 0, len(ctxFields)+len(fields)*2)
	result = append(result, ctxFields...)
	result = append(result, convertFields(fields)...)

	return setContextFields(ctx, result)
}

func (l logger) Debug(ctx context.Context, str string) {
	l.withContextFields(ctx).Debug(str)
}

func (l logger) Info(ctx context.Context, str string) {
	l.withContextFields(ctx).Info(str)
}

func (l logger) Warn(ctx context.Context, str string) {
	l.withContextFields(ctx).Warn(str)
}

func (l logger) Error(ctx context.Context, str string) {
	l.withContextFields(ctx).Error(str)
}

func (l logger) Log(ctx context.Context, level Level, str string) {
	if level == LevelDisabled {
		return
	}

	l.withContextFields(ctx).Log(ctx, slogLevelMap[level], str)
}

func (l logger) withContextFields(ctx context.Context) *slog.Logger {
	return l.impl.With(getContextFields(ctx)...)
}

func getContextFields(ctx context.Context) []any {
	fields, _ := ctx.Value(fieldsContextKey).([]any)
	return fields
}

func setContextFields(ctx context.Context, fields []any) context.Context {
	if len(fields) == 0 {
		return ctx
	}

	return context.WithValue(ctx, fieldsContextKey, fields)
}

func convertFields(fields Fields) []any {
	if len(fields) == 0 {
		return nil
	}

	result := make([]any, 0, len(fields)*2)
	for key, value := range fields {
		result = append(result, key, value)
	}

	return result
}
