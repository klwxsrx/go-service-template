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
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var zerologLevelMap = map[Level]zerolog.Level{
	LevelDebug: zerolog.DebugLevel,
	LevelInfo:  zerolog.InfoLevel,
	LevelWarn:  zerolog.WarnLevel,
	LevelError: zerolog.ErrorLevel,
}

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
	Log(ctx context.Context, level Level, str string)
}

type logger struct {
	impl   zerolog.Logger
	fields Fields
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
	l.impl = l.impl.With().Stack().Err(err).Logger()
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
	l.loggerWithFields(ctx).Debug().Msg(str)
}

func (l logger) Error(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Error().Msg(str)
}

func (l logger) Warn(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Warn().Msg(str)
}

func (l logger) Info(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Info().Msg(str)
}

func (l logger) Fatal(ctx context.Context, str string) {
	l.loggerWithFields(ctx).Fatal().Msg(str)
}

func (l logger) Log(ctx context.Context, level Level, str string) {
	l.loggerWithFields(ctx).WithLevel(zerologLevelMap[level]).Msg(str)
}

func (l logger) loggerWithFields(ctx context.Context) *zerolog.Logger {
	fields := l.fields
	ctxFields := getFieldsFromContextOrNil(ctx)
	if len(ctxFields) > 0 {
		fields = deepCopyFields(l.fields, len(ctxFields))
		mergeFields(fields, ctxFields)
	}

	z := l.impl.With().Fields(map[string]any(fields)).Logger()
	return &z
}

func New(lvl Level) Logger {
	z := zerolog.New(os.Stdout).Level(zerologLevelMap[lvl]).With().Timestamp().Logger()
	return logger{
		impl:   z,
		fields: make(Fields),
	}
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
