//go:generate ${TOOLS_BIN}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Logger=Logger"
package log

import (
	"context"
	"log/slog"
	"os"
)

type contextKey int

const (
	fieldsContextKey contextKey = iota
)

type Level int

const (
	LevelDisabled Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

var slogLevelMap = map[Level]slog.Level{
	LevelDebug: slog.LevelDebug,
	LevelInfo:  slog.LevelInfo,
	LevelWarn:  slog.LevelWarn,
	LevelError: slog.LevelError,
}

type Fields map[string]any

type Logger interface {
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

type logger struct {
	impl   *slog.Logger
	fields Fields
}

func New(level Level) Logger {
	if level == LevelDisabled {
		return stub{}
	}

	return logger{
		impl: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slogLevelMap[level],
		})),
		fields: make(Fields),
	}
}

func (l logger) With(fields Fields) Logger {
	if len(fields) == 0 {
		return l
	}

	newFields := deepCopyFields(l.fields, len(fields))
	mergeFields(newFields, fields)
	l.fields = newFields
	return l
}

func (l logger) WithField(name string, v any) Logger {
	newFields := deepCopyFields(l.fields, 1)
	mergeField(newFields, name, v)
	l.fields = newFields
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

	ctxFields, ok := getFieldsFromContext(ctx)
	if !ok {
		ctxFields = make(Fields, len(fields))
	} else {
		ctxFields = deepCopyFields(ctxFields, len(fields))
	}

	mergeFields(ctxFields, fields)
	return context.WithValue(ctx, fieldsContextKey, ctxFields)
}

func (l logger) Debug(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Debug(str)
}

func (l logger) Info(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Info(str)
}

func (l logger) Warn(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Warn(str)
}

func (l logger) Error(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Error(str)
}

func (l logger) Log(ctx context.Context, level Level, str string) {
	if level == LevelDisabled {
		return
	}

	l.loggerWithFields(ctx).Log(ctx, slogLevelMap[level], str)
}

func (l logger) loggerWithFields(ctx context.Context) *slog.Logger {
	fields := l.fields
	ctxFields := getFieldsFromContextOrNil(ctx)
	if len(ctxFields) > 0 {
		fields = deepCopyFields(l.fields, len(ctxFields))
		mergeFields(fields, ctxFields)
	}
	if len(fields) == 0 {
		return l.impl
	}

	impl := l.impl
	for name, value := range fields {
		impl = impl.With(name, value)
	}
	return impl
}

func mergeFields(fields, additional Fields) {
	for fieldName, fieldValue := range additional {
		mergeField(fields, fieldName, fieldValue)
	}
}

func mergeField(fields Fields, fieldName string, fieldValue any) {
	existedFieldValue, ok := fields[fieldName]
	if !ok {
		fields[fieldName] = fieldValue
		return
	}

	existedFieldValueMap, ok := existedFieldValue.(Fields)
	if !ok {
		fields[fieldName] = fieldValue
		return
	}
	var fieldValueMap Fields
	if fieldValueMap, ok = fieldValue.(Fields); !ok {
		if fieldValueMap, ok = fieldValue.(map[string]any); !ok {
			fields[fieldName] = fieldValue
			return
		}
	}

	for nestedFieldName, nestedFieldValue := range fieldValueMap {
		mergeField(existedFieldValueMap, nestedFieldName, nestedFieldValue)
	}
	fields[fieldName] = existedFieldValueMap
}

func getFieldsFromContextOrNil(ctx context.Context) Fields {
	fields, ok := getFieldsFromContext(ctx)
	if !ok {
		return nil
	}
	return fields
}

func getFieldsFromContext(ctx context.Context) (Fields, bool) {
	fields, ok := ctx.Value(fieldsContextKey).(Fields)
	return fields, ok
}

func deepCopyFields(fields Fields, additionalCapacity int) Fields {
	newFields := make(Fields, len(fields)+additionalCapacity)
	for fieldName, fieldValue := range fields {
		fieldsValue, ok := fieldValue.(Fields)
		if ok {
			fieldValue = deepCopyFields(fieldsValue, 0)
		}
		newFields[fieldName] = fieldValue
	}
	return newFields
}
