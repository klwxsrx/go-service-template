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

	Bus interface {
		Produce(ctx context.Context, messageClass string, msgs []StructuredMessage, scheduleAt time.Time) error
		RegisterMessages(msg RegisterStructuredMessageFunc, msgs ...RegisterStructuredMessageFunc) error
	}

	BusOption           func() (BusProducerMW, MetadataBuilderFunc)
	BusProducerMW       func(BusProducerFunc) BusProducerFunc
	BusProducerFunc     func(ctx context.Context, domainName string, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error
	MetadataBuilderFunc func(context.Context) (Metadata, error)
	Metadata            map[string]any
)

type bus struct {
	domainName   string
	serializer   *jsonSerializer
	producerImpl BusProducerFunc
}

func NewBus(
	domainName string,
	storage OutboxStorage,
	opts ...BusOption,
) Bus {
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
	producerImpl := newBusProducerImpl(domainName, serializer, storage, metadataBuilders)
	for i := len(producerMWs) - 1; i >= 0; i-- {
		producerImpl = producerMWs[i](producerImpl)
	}

	return bus{
		domainName:   domainName,
		serializer:   serializer,
		producerImpl: producerImpl,
	}
}

func (b bus) Produce(ctx context.Context, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error {
	if len(msgs) == 0 {
		return nil
	}

	return b.producerImpl(
		ctx,
		b.domainName,
		msgClass,
		msgs,
		scheduleAt,
	)
}

func (b bus) RegisterMessages(message RegisterStructuredMessageFunc, messages ...RegisterStructuredMessageFunc) error {
	messages = append([]RegisterStructuredMessageFunc{message}, messages...)
	for _, registerFunc := range messages {
		messageClass, messageType, topicBuilder, keyBuilder, err := registerFunc(b.domainName)
		if err != nil {
			return fmt.Errorf("register message for domain %s: %w", b.domainName, err)
		}

		err = b.serializer.RegisterSerializer(b.domainName, messageClass, messageType, topicBuilder, keyBuilder)
		if err != nil {
			return fmt.Errorf("register message serializer for message class %s type %s, domain %s: %w", messageClass, messageType, b.domainName, err)
		}
	}

	return nil
}

func newBusProducerImpl(
	domainName string,
	serializer *jsonSerializer,
	storage OutboxStorage,
	metadataBuilders []MetadataBuilderFunc,
) BusProducerFunc {
	serializeImpl := func(ctx context.Context, msgClass string, msg StructuredMessage) (*Message, error) {
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

	return func(ctx context.Context, _ string, msgClass string, msgs []StructuredMessage, scheduleAt time.Time) error {
		serializedMsgs := make([]Message, 0, len(msgs))
		for _, msg := range msgs {
			serializedMsg, err := serializeImpl(ctx, msgClass, msg)
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
				loggerWithFields.WithError(err).Log(ctx, errorLevel, "messages didn't stored due error")
			} else {
				loggerWithFields.Log(ctx, infoLevel, "messages stored")
			}

			return err
		}
	}

	return func() (BusProducerMW, MetadataBuilderFunc) {
		return loggingMW, nil
	}
}

type BusFactory struct {
	storage OutboxStorage
	opts    []BusOption
}

func NewBusFactory(
	storage OutboxStorage,
	opts ...BusOption,
) BusFactory {
	return BusFactory{
		storage: storage,
		opts:    opts,
	}
}

func (f BusFactory) New(domainName string) Bus {
	return NewBus(
		domainName,
		f.storage,
		f.opts...,
	)
}
