package message

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

type (
	TopicMessagesMap map[Topic]TopicMessages
	TopicMessages    []RegisterMessageFunc

	ProducerRegistry interface {
		RegisterMessages(TopicMessagesMap) error
	}

	BusProducer interface {
		Produce(context.Context, []StructuredMessage, time.Time) error
		ProducerRegistry
	}

	RegisterMessageFunc func() (StructuredMessage, KeyBuilderFunc)
	KeyBuilderFunc      func(StructuredMessage) string

	BusOption           func() (BusProducerMW, MetadataBuilderFunc)
	BusProducerMW       func(BusProducerFunc) BusProducerFunc
	BusProducerFunc     func(context.Context, []StructuredMessage, time.Time) error
	MetadataBuilderFunc func(context.Context) (Metadata, error)
)

type busProducer struct {
	topicSerializers map[Topic]jsonSerializer
	messageTopics    map[reflect.Type][]Topic
	producerImpl     BusProducerFunc
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

	topicSerializers := make(map[Topic]jsonSerializer)
	messageTopics := make(map[reflect.Type][]Topic)

	producerImpl := newBusProducerImpl(topicSerializers, messageTopics, metadataBuilders, storage)
	for i := len(producerMWs) - 1; i >= 0; i-- {
		producerImpl = producerMWs[i](producerImpl)
	}

	return busProducer{
		topicSerializers: topicSerializers,
		messageTopics:    messageTopics,
		producerImpl:     producerImpl,
	}
}

func (b busProducer) Produce(ctx context.Context, msgs []StructuredMessage, scheduleAt time.Time) error {
	if len(msgs) == 0 {
		return nil
	}

	now := time.Now()
	if scheduleAt.Before(now) {
		scheduleAt = now
	}

	return b.producerImpl(
		ctx,
		msgs,
		scheduleAt,
	)
}

func (b busProducer) RegisterMessages(topicMessages TopicMessagesMap) error {
	for topic, msgs := range topicMessages {
		for _, registerFunc := range msgs {
			msgSchema, keyBuilder := registerFunc()
			msgSchemaType := reflect.TypeOf(msgSchema)

			topics, ok := b.messageTopics[msgSchemaType]
			if !ok {
				topics = make([]Topic, 0, 1)
			}

			topics = append(topics, topic)
			b.messageTopics[msgSchemaType] = topics

			serializer, ok := b.topicSerializers[topic]
			if !ok {
				serializer = newJSONSerializer(topic)
				b.topicSerializers[topic] = serializer
			}

			messageType := msgSchema.Type()
			if messageType == "" {
				return fmt.Errorf("get message type for %T: blank message must return const value", msgSchema)
			}

			err := serializer.RegisterSerializer(messageType, keyBuilder)
			if err != nil {
				return fmt.Errorf("register message serializer for type %s: %w", messageType, err)
			}
		}
	}

	return nil
}

func newBusProducerImpl(
	topicSerializers map[Topic]jsonSerializer,
	messageTopics map[reflect.Type][]Topic,
	metadataBuilders []MetadataBuilderFunc,
	storage OutboxStorage,
) BusProducerFunc {
	serializeImpl := func(ctx context.Context, msg StructuredMessage, serializer jsonSerializer) (*Message, error) {
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

		serializedMsg, err := serializer.Serialize(msg, meta)
		if err != nil {
			return nil, fmt.Errorf("serialize message %T: %w", msg, err)
		}

		return serializedMsg, nil
	}

	return func(ctx context.Context, msgs []StructuredMessage, scheduleAt time.Time) error {
		serializedMsgs := make([]Message, 0, len(msgs))
		for _, msg := range msgs {
			topics, ok := messageTopics[reflect.TypeOf(msg)]
			if !ok {
				return fmt.Errorf("unknown message type %T", msg)
			}

			for _, topic := range topics {
				serializer, ok := topicSerializers[topic]
				if !ok {
					return fmt.Errorf("serializer for topic %s not found", topic)
				}

				serializedMsg, err := serializeImpl(ctx, msg, serializer)
				if err != nil {
					return err
				}

				serializedMsgs = append(serializedMsgs, *serializedMsg)
			}
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
		return func(ctx context.Context, msgs []StructuredMessage, scheduleAt time.Time) error {
			err := impl(ctx, msgs, scheduleAt)

			metrics.WithLabel("success", err == nil).Count("msg_store_attempts_total", len(msgs))
			return err
		}
	}

	return func() (BusProducerMW, MetadataBuilderFunc) {
		return metricsMW, nil
	}
}

func WithLogging(logger log.Logger, infoLevel, errorLevel log.Level) BusOption {
	loggingMW := func(impl BusProducerFunc) BusProducerFunc {
		return func(ctx context.Context, msgs []StructuredMessage, scheduleAt time.Time) error {
			messageIDTypes := make([]log.Fields, 0, len(msgs))
			for _, msg := range msgs {
				messageIDTypes = append(messageIDTypes, log.Fields{
					"id":   msg.ID(),
					"type": msg.Type(),
				})
			}

			loggerWithFields := logger.With(log.Fields{
				"messageIDType": messageIDTypes,
				"scheduleAt":    scheduleAt,
			})

			err := impl(ctx, msgs, scheduleAt)
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
