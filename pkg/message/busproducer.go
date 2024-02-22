package message

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

const requestIDMetadataKey = "requestID"

type (
	StructuredMessage interface {
		ID() uuid.UUID
		// Type must be unique string for message class
		Type() string
	}

	RegisterStructuredMessageFunc func(
		domainName string,
	) (messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc, err error)

	BusProducer interface {
		Produce(ctx context.Context, domainName, messageClass string, msgs []StructuredMessage, scheduleAt time.Time) error
		RegisterMessages(domainName string, msg RegisterStructuredMessageFunc, msgs ...RegisterStructuredMessageFunc) error
	}

	BusOption           func() (BusProducerMW, MetadataBuilderFunc)
	BusProducerMW       func(BusProducerFunc) BusProducerFunc
	BusProducerFunc     func(ctx context.Context, domainName string, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error
	MetadataBuilderFunc func(context.Context) (Metadata, error)
	Metadata            map[string]any
)

type busProducer struct {
	serializer   *jsonSerializer
	producerImpl BusProducerFunc
}

func NewBusProducer(
	storage OutboxStorage,
	opts ...BusOption,
) BusProducer {
	producerMWs := make([]BusProducerMW, 0, len(opts))
	metadataBuilders := make([]MetadataBuilderFunc, 0)
	for _, opt := range opts {
		producerMW, metaBuilder := opt()
		if producerMW != nil {
			producerMWs = append(producerMWs, producerMW)
		}
		if metaBuilder != nil {
			metadataBuilders = append(metadataBuilders, metaBuilder)
		}
	}

	serializer := newJSONSerializer()
	producerImpl := newBusProducerImpl(serializer, storage, metadataBuilders)
	for i := len(producerMWs) - 1; i >= 0; i-- {
		producerImpl = producerMWs[i](producerImpl)
	}

	return busProducer{
		serializer:   serializer,
		producerImpl: producerImpl,
	}
}

func (b busProducer) Produce(ctx context.Context, domainName, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error {
	if len(msgs) == 0 {
		return nil
	}

	now := time.Now()
	if scheduleAt.Before(now) {
		scheduleAt = now
	}

	return b.producerImpl(
		ctx,
		domainName,
		msgClass,
		msgs,
		scheduleAt,
	)
}

func (b busProducer) RegisterMessages(domainName string, message RegisterStructuredMessageFunc, messages ...RegisterStructuredMessageFunc) error {
	messages = append([]RegisterStructuredMessageFunc{message}, messages...)
	for _, registerFunc := range messages {
		messageClass, messageType, topicBuilder, keyBuilder, err := registerFunc(domainName)
		if err != nil {
			return fmt.Errorf("register message for domain %s: %w", domainName, err)
		}

		err = b.serializer.RegisterSerializer(domainName, messageClass, messageType, topicBuilder, keyBuilder)
		if err != nil {
			return fmt.Errorf("register message serializer for message class %s type %s, domain %s: %w", messageClass, messageType, domainName, err)
		}
	}

	return nil
}

func newBusProducerImpl(
	serializer *jsonSerializer,
	storage OutboxStorage,
	metadataBuilders []MetadataBuilderFunc,
) BusProducerFunc {
	serializeImpl := func(ctx context.Context, domainName, msgClass string, msg StructuredMessage) (*Message, error) {
		meta := make(Metadata, len(metadataBuilders))
		for _, metaBuilder := range metadataBuilders {
			tmpMeta, err := metaBuilder(ctx)
			if err != nil {
				return nil, fmt.Errorf("build metadata for %T: %w", msg, err)
			}
			for key, value := range tmpMeta {
				meta[key] = value
			}
		}

		serializedMsg, err := serializer.Serialize(domainName, msgClass, msg, meta)
		if err != nil {
			return nil, fmt.Errorf("serialize message %T: %w", msg, err)
		}

		return serializedMsg, nil
	}

	return func(ctx context.Context, domainName, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error {
		serializedMsgs := make([]Message, 0, len(msgs))
		for _, msg := range msgs {
			serializedMsg, err := serializeImpl(ctx, domainName, msgClass, msg)
			if err != nil {
				return err
			}

			serializedMsgs = append(serializedMsgs, *serializedMsg)
		}

		return storage.Store(ctx, serializedMsgs, scheduleAt)
	}
}

func WithObservability(observer observability.Observer) BusOption {
	observabilityMetadataBuilder := func(ctx context.Context) (Metadata, error) { // nolint:unparam
		requestID, ok := observer.RequestID(ctx)
		if !ok {
			return nil, nil
		}

		return Metadata{requestIDMetadataKey: requestID}, nil
	}

	return func() (BusProducerMW, MetadataBuilderFunc) {
		return nil, observabilityMetadataBuilder
	}
}

func WithMetrics(metrics metric.Metrics) BusOption {
	metricsMW := func(impl BusProducerFunc) BusProducerFunc {
		return func(ctx context.Context, domainName string, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error {
			err := impl(ctx, domainName, msgClass, msgs, scheduleAt)

			metricsWithLabels := metrics.With(metric.Labels{
				"domain":  domainName,
				"class":   msgClass,
				"success": err == nil,
			})
			for _, msg := range msgs {
				metricsWithLabels.WithLabel("type", msg.Type()).Increment("msg_store_attempts_total")
			}

			return err
		}
	}

	return func() (BusProducerMW, MetadataBuilderFunc) {
		return metricsMW, nil
	}
}

func WithLogging(logger log.Logger, infoLevel, errorLevel log.Level) BusOption {
	loggingMW := func(impl BusProducerFunc) BusProducerFunc {
		return func(ctx context.Context, domainName string, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error {
			messageIDTypes := make([]log.Fields, 0, len(msgs))
			for _, msg := range msgs {
				messageIDTypes = append(messageIDTypes, log.Fields{
					"id":   msg.ID(),
					"type": msg.Type(),
				})
			}

			loggerWithFields := logger.With(log.Fields{
				"domainName":    domainName,
				"messageClass":  msgClass,
				"messageIDType": messageIDTypes,
				"scheduleAt":    scheduleAt,
			})

			err := impl(ctx, domainName, msgClass, msgs, scheduleAt)
			if err != nil {
				loggerWithFields.WithError(err).Log(ctx, errorLevel, "messages didn't stored to outbox storage due error")
			} else {
				loggerWithFields.Log(ctx, infoLevel, "messages stored to outbox storage")
			}

			return err
		}
	}

	return func() (BusProducerMW, MetadataBuilderFunc) {
		return loggingMW, nil
	}
}
