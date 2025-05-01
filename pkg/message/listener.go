package message

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/idk"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type HandlerMiddleware func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage]

type listener struct {
	consumer     Consumer
	deserializer jsonDeserializer
	handler      TypedHandler[StructuredMessage]
}

func newListener(
	consumer Consumer,
	messageDeserializer jsonDeserializer,
	handler TypedHandler[StructuredMessage],
	mws ...HandlerMiddleware,
) worker.ErrorJob {
	l := &listener{
		consumer:     consumer,
		deserializer: messageDeserializer,
		handler:      handler,
	}

	l.handler = l.wrapWithPanicHandler(l.handler)
	for i := len(mws) - 1; i >= 0; i-- {
		l.handler = mws[i](l.handler)
	}

	return l.workerImpl
}

func (l *listener) wrapWithPanicHandler(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
	return func(ctx context.Context, msg StructuredMessage) (err error) {
		recoverPanic := func(ctx context.Context) {
			panicMsg := recover()
			if panicMsg == nil {
				return
			}

			meta := getHandlerMetadata(ctx)
			meta.Panic = &panicErr{
				Message:    fmt.Sprintf("%v", panicMsg),
				Stacktrace: debug.Stack(),
			}

			err = fmt.Errorf("message handled with panic: %v", panicMsg)
		}

		defer recoverPanic(ctx)
		return handler(ctx, msg)
	}
}

func (l *listener) workerImpl(ctx context.Context) error {
	err := func(ctx context.Context) error {
		for {
			select {
			case msg, ok := <-l.consumer.Messages():
				if !ok {
					return errors.New("consumer closed messages channel")
				}
				l.processMessage(msg)
			case <-ctx.Done():
				return l.consumer.Close()
			}
		}
	}(ctx)
	if err != nil {
		return fmt.Errorf("message listener %s: %w", l.consumer.Name(), err)
	}

	return nil
}

func (l *listener) processMessage(msg *ConsumerMessage) {
	deserializedMsg, meta, err := l.deserializer.Deserialize(msg.Message.Payload)
	if errors.Is(err, errDeserializeNotValidMessage) || errors.Is(err, errDeserializeUnknownMessage) {
		l.consumer.Ack(msg)
		return
	}
	if err != nil {
		l.consumer.Nack(msg)
		return
	}

	ctx := withHandlerMetadata(msg.Context, &msg.Message, meta)
	err = l.handler(ctx, deserializedMsg)
	if err != nil {
		l.consumer.Nack(msg)
		return
	}

	l.consumer.Ack(msg)
}

func WithHandlerIdempotencyKeyErrorIgnoring() HandlerMiddleware {
	return WithHandlerErrorMapping(func(err error) error {
		if errors.Is(err, idk.ErrAlreadyInserted) {
			return nil
		}

		return err
	})
}

func WithHandlerErrorMapping(fn func(error) error) HandlerMiddleware {
	return func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			err := handler(ctx, msg)
			return fn(err)
		}
	}
}

func WithHandlerLogging(logger log.Logger, infoLevel, errorLevel log.Level) HandlerMiddleware {
	return func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			meta := getHandlerMetadata(ctx)
			ctx = logger.WithContext(ctx, log.Fields{
				"consumerMessage": log.Fields{
					"correlationID": uuid.New(),
					"topic":         meta.MessageTopic,
					"messageID":     meta.MessageID,
					"messageType":   msg.Type(),
				},
			})

			err := handler(ctx, msg)
			if meta.Panic != nil {
				logger.WithField("panic", log.Fields{
					"message": meta.Panic.Message,
					"stack":   string(meta.Panic.Stacktrace),
				}).Error(ctx, "message handled with panic")
				return err
			}
			if err != nil {
				logger.WithError(err).Log(ctx, errorLevel, "message handled with error")
				return err
			}

			logger.Log(ctx, infoLevel, "message handled")
			return nil
		}
	}
}

func WithHandlerMetrics(metrics metric.Metrics) HandlerMiddleware {
	return func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			started := time.Now()

			err := handler(ctx, msg)
			meta := getHandlerMetadata(ctx)
			if meta.Panic != nil {
				metrics.WithLabel("topic", meta.MessageTopic).Increment("msg_handle_panics_total")
			}

			metrics.With(metric.Labels{
				"topic":   meta.MessageTopic,
				"success": err == nil,
			}).Duration("msg_handle_duration_seconds", time.Since(started))
			return err
		}
	}
}

func WithHandlerObservability(observer observability.Observer, fields ...observability.Field) HandlerMiddleware {
	if len(fields) == 0 {
		return func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
			return func(ctx context.Context, msg StructuredMessage) error {
				return handler(ctx, msg)
			}
		}
	}

	return func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			meta := getHandlerMetadata(ctx)
			observabilityValues, ok := meta.MessageMetadata[observabilityMetadataKey].(map[string]any)
			if !ok {
				return handler(ctx, msg)
			}

			for _, field := range fields {
				value, ok := observabilityValues[string(field)].(string)
				if ok && value != "" {
					ctx = observer.WithField(ctx, field, value)
				}
			}

			return handler(ctx, msg)
		}
	}
}
