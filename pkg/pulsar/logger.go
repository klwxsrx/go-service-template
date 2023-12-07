package pulsar

import (
	"context"
	"fmt"

	pulsarlog "github.com/apache/pulsar-client-go/pulsar/log"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

type loggerAdapter struct {
	ctx    context.Context
	logger log.Logger
}

func newLoggerAdapter(logger log.Logger) pulsarlog.Logger {
	return loggerAdapter{context.Background(), logger}
}

func (l loggerAdapter) SubLogger(fields pulsarlog.Fields) pulsarlog.Logger {
	return &loggerAdapter{l.ctx, l.logger.With(log.Fields(fields))}
}

func (l loggerAdapter) WithFields(fields pulsarlog.Fields) pulsarlog.Entry {
	return &loggerAdapter{l.ctx, l.logger.With(log.Fields(fields))}
}

func (l loggerAdapter) WithField(name string, value any) pulsarlog.Entry {
	return &loggerAdapter{l.ctx, l.logger.WithField(name, value)}
}

func (l loggerAdapter) WithError(err error) pulsarlog.Entry {
	return &loggerAdapter{l.ctx, l.logger.WithError(err)}
}

func (l loggerAdapter) Debug(args ...any) {
	l.logger.Debug(l.ctx, fmt.Sprint(args...))
}

func (l loggerAdapter) Info(args ...any) {
	l.logger.Info(l.ctx, fmt.Sprint(args...))
}

func (l loggerAdapter) Warn(args ...any) {
	l.logger.Warn(l.ctx, fmt.Sprint(args...))
}

func (l loggerAdapter) Error(args ...any) {
	l.logger.Error(l.ctx, fmt.Sprint(args...))
}

func (l loggerAdapter) Debugf(format string, args ...any) {
	l.logger.Debug(l.ctx, fmt.Sprintf(format, args...))
}

func (l loggerAdapter) Infof(format string, args ...any) {
	l.logger.Info(l.ctx, fmt.Sprintf(format, args...))
}

func (l loggerAdapter) Warnf(format string, args ...any) {
	l.logger.Warn(l.ctx, fmt.Sprintf(format, args...))
}

func (l loggerAdapter) Errorf(format string, args ...any) {
	l.logger.Error(l.ctx, fmt.Sprintf(format, args...))
}
