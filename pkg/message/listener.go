package message

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/idk"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
	"github.com/klwxsrx/go-service-template/pkg/worker"
)

const defaultWorkersCount = 1

type (
	ListenerImpl struct {
		// MaxProcessedMessages is the max number of simultaneously processed messages
		MaxProcessedMessages     int
		Middlewares              []HandlerMiddleware
		HandlerRetry             backoff.BackOff
		Workers                  worker.Pool
		OnHandlerNotFound        []func(context.Context, *Message)
		OnBeforeHandleMessage    []func(context.Context, *Message) context.Context
		OnHandlerResult          []func(context.Context, *Message, error)
		OnAcknowledgeResult      []func(_ context.Context, _ *Message, handlerResult error, ackErr error)
		OnDeserializedUnknownMsg []func(context.Context, *Message, error)
		OnDeserializedError      []func(context.Context, *Message, error)

		consumer     Consumer[any]
		handlers     map[string][]TypedHandler[StructuredMessage]
		deserializer Deserializer
		queue        ListenerProcessingQueue
		queueRetry   backoff.BackOff
	}

	ListenerQueueBuilder[S AcknowledgeStrategy] func(ackStrategy S, queueSize int) ListenerProcessingQueue

	ListenerOption    func(*ListenerImpl)
	HandlerMiddleware func(TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage]
)

func NewListener[S AcknowledgeStrategy](
	consumer Consumer[S],
	messageHandlers map[string][]TypedHandler[StructuredMessage],
	processingQueue ListenerQueueBuilder[S],
	deserializer Deserializer,
	opts ...ListenerOption,
) worker.ErrorJob {
	defaultHandlerRetry := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(100*time.Millisecond),
		backoff.WithMultiplier(2),
		backoff.WithMaxInterval(5*time.Minute),
		backoff.WithMaxElapsedTime(0),
	)

	defaultQueueRetry := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(100*time.Millisecond),
		backoff.WithMultiplier(2),
		backoff.WithMaxInterval(time.Minute),
		backoff.WithMaxElapsedTime(0),
	)

	impl := &ListenerImpl{
		MaxProcessedMessages:     defaultWorkersCount,
		Middlewares:              nil,
		HandlerRetry:             defaultHandlerRetry,
		Workers:                  worker.NewPool(defaultWorkersCount),
		OnHandlerNotFound:        nil,
		OnBeforeHandleMessage:    nil,
		OnHandlerResult:          nil,
		OnAcknowledgeResult:      nil,
		OnDeserializedUnknownMsg: nil,
		OnDeserializedError:      nil,

		consumer:     consumerAdapter[S]{consumer},
		handlers:     messageHandlers,
		deserializer: deserializer,
		queueRetry:   defaultQueueRetry,
	}
	for _, opt := range opts {
		opt(impl)
	}

	for msgType, handlers := range impl.handlers {
		for i := range handlers {
			handlers[i] = impl.wrapWithPanicHandler(handlers[i])
			for j := len(impl.Middlewares) - 1; j >= 0; j-- {
				handlers[i] = impl.Middlewares[j](handlers[i])
			}
		}
		impl.handlers[msgType] = handlers
	}

	impl.queue = processingQueue(
		consumer.Acknowledge(),
		impl.MaxProcessedMessages,
	)

	return impl.consumerWorker
}

func (l *ListenerImpl) wrapWithPanicHandler(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
	return func(ctx context.Context, msg StructuredMessage) (err error) {
		recoverPanic := func(ctx context.Context) {
			panicMsg := recover()
			if panicMsg == nil {
				return
			}

			meta := GetHandlerMetadata(ctx)
			meta.Panic = &PanicErr{
				Message:    fmt.Sprintf("%v", panicMsg),
				Stacktrace: debug.Stack(),
			}

			err = fmt.Errorf("message handled with panic: %v", panicMsg)
		}

		defer recoverPanic(ctx)
		return handler(ctx, msg)
	}
}

func (l *ListenerImpl) consumerWorker(ctx context.Context) error {
	err := func() error {
		wg := &sync.WaitGroup{}
		defer wg.Wait()

		for {
			select {
			case <-l.queue.ProcessingTokens():
			case <-ctx.Done():
				return l.consumer.Close()
			}

			select {
			case msg, ok := <-l.consumer.Messages():
				if !ok {
					return errors.New("consumer closed messages channel")
				}
				if err := l.queue.AddProcessing(msg); err != nil {
					return fmt.Errorf("add to processing internal error: %w", err)
				}

				wg.Add(1)
				go l.processMessage(ctx, msg, wg)
			case <-ctx.Done():
				return l.consumer.Close()
			}
		}
	}()
	if err != nil {
		return fmt.Errorf("message listener %s/%s: %w", l.consumer.Subscriber(), l.consumer.Topic(), err)
	}

	return nil
}

func (l *ListenerImpl) processMessage(ctx context.Context, msg *ConsumerMessage, processing *sync.WaitGroup) {
	skipAndAckMessage := func(ctx context.Context, msg *ConsumerMessage) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := l.acknowledgeMessage(ctx, ctx, msg, nil); err == nil {
				break
			}
		}
	}

	defer processing.Done()

	msgImpl, meta, err := l.deserializer.Deserialize(msg.Message.Payload)
	if errors.Is(err, ErrDeserializeUnknownMessage) {
		for _, fn := range l.OnDeserializedUnknownMsg {
			fn(ctx, &msg.Message, err)
		}
		skipAndAckMessage(ctx, msg)
		return
	}
	if err != nil {
		for _, fn := range l.OnDeserializedError {
			fn(ctx, &msg.Message, err)
		}
		skipAndAckMessage(ctx, msg)
		return
	}

	handlers, ok := l.handlers[msgImpl.Type()]
	if !ok {
		for _, fn := range l.OnHandlerNotFound {
			fn(ctx, &msg.Message)
		}
		skipAndAckMessage(ctx, msg)
		return
	}

	msgCtx := withHandlerMetadata(msg.Context, &msg.Message, meta)
	for _, fn := range l.OnBeforeHandleMessage {
		msgCtx = fn(msgCtx, &msg.Message)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		handlersGroup := worker.WithinFailSafeGroup(msgCtx, l.Workers)
		for _, handler := range handlers {
			handlersGroup.Do(func(msgCtx context.Context) error {
				return backoff.Retry(
					func() error { return handler(msgCtx, msgImpl) },
					backoff.WithContext(l.HandlerRetry, ctx),
				)
			})
		}

		handlerErr := handlersGroup.Wait()
		for _, fn := range l.OnHandlerResult {
			fn(msgCtx, &msg.Message, handlerErr)
		}

		if err = l.acknowledgeMessage(ctx, msgCtx, msg, handlerErr); err == nil {
			break
		}
	}
}

func (l *ListenerImpl) acknowledgeMessage(ctx, msgCtx context.Context, msg *ConsumerMessage, handlerErr error) error {
	return backoff.Retry(
		func() error {
			if handlerErr != nil && errors.Is(handlerErr, ctx.Err()) {
				return ctx.Err()
			}

			err := l.queue.AcknowledgeResult(ctx, msg, handlerErr)
			for _, fn := range l.OnAcknowledgeResult {
				fn(msgCtx, &msg.Message, handlerErr, err)
			}
			if errors.Is(err, ErrNegativeAckNotSupported) {
				return backoff.Permanent(err)
			}

			return err
		},
		backoff.WithContext(l.queueRetry, ctx),
	)
}

func WithHandlerIdempotencyKeyErrorIgnoring() ListenerOption {
	return WithHandlerErrorMapping(func(err error) error {
		if errors.Is(err, idk.ErrAlreadyInserted) {
			return nil
		}

		return err
	})
}

func WithHandlerErrorMapping(fn func(error) error) ListenerOption {
	mw := func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			err := handler(ctx, msg)
			return fn(err)
		}
	}

	return func(l *ListenerImpl) {
		l.Middlewares = append(l.Middlewares, mw)
	}
}

func WithHandlerLogging(logger log.Logger, infoLevel, errorLevel log.Level) ListenerOption {
	mw := func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			meta := GetHandlerMetadata(ctx)
			ctx = logger.WithContext(ctx, log.Fields{
				"consumerMessage": log.Fields{
					"correlation": uuid.New(),
					"topic":       meta.MessageTopic,
					"messageID":   meta.MessageID,
					"messageType": msg.Type(),
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

	return func(l *ListenerImpl) {
		l.Middlewares = append(l.Middlewares, mw)

		l.OnHandlerNotFound = append(l.OnHandlerNotFound, func(ctx context.Context, msg *Message) {
			logger.With(log.Fields{
				"messageID": msg.ID,
				"topic":     msg.Topic,
			}).Log(ctx, errorLevel, "message handler not found")
		})

		l.OnBeforeHandleMessage = append(l.OnBeforeHandleMessage, func(ctx context.Context, _ *Message) context.Context {
			return logger.WithContext(ctx, log.Fields{"handlerCorrelation": uuid.New().String()})
		})

		l.OnAcknowledgeResult = append(l.OnAcknowledgeResult, func(ctx context.Context, msg *Message, handlerResult, err error) {
			if err == nil {
				return
			}

			var handlerResultStr *string
			if handlerResult != nil {
				v := handlerResult.Error()
				handlerResultStr = &v
			}

			logger.
				With(log.Fields{
					"messageID":    msg.ID,
					"topic":        msg.Topic,
					"handleResult": handlerResultStr,
				}).
				WithError(err).
				Log(ctx, errorLevel, "failed to acknowledge handled message")
		})

		l.OnDeserializedUnknownMsg = append(l.OnDeserializedUnknownMsg, func(ctx context.Context, msg *Message, err error) {
			logger.
				With(log.Fields{
					"messageID": msg.ID,
					"topic":     msg.Topic,
				}).
				WithError(err).
				Log(ctx, errorLevel, "failed to deserialize message")
		})

		l.OnDeserializedError = append(l.OnDeserializedError, func(ctx context.Context, msg *Message, err error) {
			logger.
				With(log.Fields{
					"messageID": msg.ID,
					"topic":     msg.Topic,
				}).
				WithError(err).
				Log(ctx, errorLevel, "failed to deserialize message")
		})
	}
}

func WithHandlerMetrics(metrics metric.Metrics) ListenerOption {
	mw := func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			started := time.Now()

			err := handler(ctx, msg)
			meta := GetHandlerMetadata(ctx)
			if meta.Panic != nil {
				metrics.With(metric.Labels{
					"topic": meta.MessageTopic,
					"type":  msg.Type(),
				}).Increment("msg_handle_panics_total")
			}

			metrics.With(metric.Labels{
				"topic":   meta.MessageTopic,
				"type":    msg.Type(),
				"success": err == nil,
			}).Duration("msg_handle_duration_seconds", time.Since(started))
			return err
		}
	}

	return func(l *ListenerImpl) {
		l.Middlewares = append(l.Middlewares, mw)
	}
}

func WithHandlerObservability(observer observability.Observer, fields ...observability.Field) ListenerOption {
	if len(fields) == 0 {
		return func(*ListenerImpl) {}
	}

	mw := func(handler TypedHandler[StructuredMessage]) TypedHandler[StructuredMessage] {
		return func(ctx context.Context, msg StructuredMessage) error {
			metadata := GetHandlerMetadata(ctx).MessageMetadata
			for _, field := range fields {
				value, ok := metadata[fmt.Sprintf("%s%s", observabilityMetaKeyPrefix, field)]
				if ok && value != "" {
					ctx = observer.WithField(ctx, field, value)
				}
			}

			return handler(ctx, msg)
		}
	}

	return func(l *ListenerImpl) {
		l.Middlewares = append(l.Middlewares, mw)
	}
}

func WithHandlerMultipleWorkers(workersCount int) ListenerOption {
	if workersCount <= worker.MaxWorkersCountNumCPU {
		workersCount = runtime.NumCPU()
	}
	if workersCount == 0 {
		workersCount = 1
	}

	return func(l *ListenerImpl) {
		l.Workers = worker.NewPool(workersCount)
		l.MaxProcessedMessages = workersCount
	}
}

func WithHandlerWorkerPool(pool worker.Pool, maxProcessedMessagesLimit int) ListenerOption {
	return func(l *ListenerImpl) {
		l.Workers = pool
		l.MaxProcessedMessages = maxProcessedMessagesLimit
	}
}

func WithHandlerRetry(retry backoff.BackOff) ListenerOption {
	return func(l *ListenerImpl) {
		l.HandlerRetry = retry
	}
}

type consumerAdapter[S AcknowledgeStrategy] struct {
	Consumer[S]
}

func (a consumerAdapter[S]) Acknowledge() any {
	return nil
}
