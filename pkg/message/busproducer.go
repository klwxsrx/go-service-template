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
	Registry interface {
		Register(TopicMessages, ...BusProducerOption) error
	}

	BusProducer interface {
		Registry
		Produce(context.Context, ...StructuredMessage) error
	}

	BusScheduledProducer interface {
		BusProducer
		Schedule(context.Context, time.Time, ...StructuredMessage) error
	}

	TopicMessages map[Topic][]RegisterMessageFunc

	BusProducerOption func(*BusProducerConfig)
	BusProducerConfig struct {
		Middlewares      []BusProducerMiddleware
		MetadataBuilders []MetadataBuilder
	}
	BusProducerMiddleware func(BusProduce) BusProduce
	MetadataBuilder       func(context.Context) (Metadata, error)
	BusProduce            func(context.Context, Topic, StructuredMessage, *time.Time) error

	busProducerImpl struct {
		producerImpl  producerImpl
		serializer    Serializer
		messageTopics map[reflect.Type][]Topic
		producers     map[producerKey]BusProduce
		topicMessages map[Topic]map[string]struct{}
		baseConfig    BusProducerConfig
	}

	producerKey struct {
		Topic   Topic
		Message reflect.Type
	}

	producerImpl func(context.Context, *Message, *time.Time) error
)

func NewBusProducer(
	producer Producer,
	serializer Serializer,
	opts ...BusProducerOption,
) BusProducer {
	config := BusProducerConfig{
		Middlewares:      nil,
		MetadataBuilders: nil,
	}
	for _, opt := range opts {
		opt(&config)
	}

	return &busProducerImpl{
		producerImpl: func(ctx context.Context, message *Message, _ *time.Time) error {
			err := producer.Produce(ctx, message)
			if err != nil {
				return fmt.Errorf("produce message: %w", err)
			}

			return nil
		},
		serializer:    serializer,
		messageTopics: make(map[reflect.Type][]Topic),
		producers:     make(map[producerKey]BusProduce),
		topicMessages: make(map[Topic]map[string]struct{}),
		baseConfig:    config,
	}
}

func NewBusScheduledProducer(
	storage Storage,
	serializer Serializer,
	opts ...BusProducerOption,
) BusScheduledProducer {
	config := BusProducerConfig{
		Middlewares:      nil,
		MetadataBuilders: nil,
	}
	for _, opt := range opts {
		opt(&config)
	}

	return &busProducerImpl{
		producerImpl: func(ctx context.Context, message *Message, at *time.Time) error {
			scheduleAt := time.Now()
			if at != nil {
				scheduleAt = *at
			}

			err := storage.Store(ctx, scheduleAt, *message)
			if err != nil {
				return fmt.Errorf("store message: %w", err)
			}

			return nil
		},
		serializer:    serializer,
		messageTopics: make(map[reflect.Type][]Topic),
		producers:     make(map[producerKey]BusProduce),
		topicMessages: make(map[Topic]map[string]struct{}),
		baseConfig:    config,
	}
}

func (p *busProducerImpl) Register(msgs TopicMessages, opts ...BusProducerOption) error {
	config := p.createFromBaseConfig()
	for _, opt := range opts {
		opt(&config)
	}

	for topic, funcs := range msgs {
		for _, fn := range funcs {
			if err := p.registerImpl(topic, fn, config); err != nil {
				return fmt.Errorf("register for topic %s: %w", topic, err)
			}
		}
	}

	return nil
}

func (p *busProducerImpl) Produce(ctx context.Context, msgs ...StructuredMessage) error {
	return p.scheduleImpl(ctx, nil, msgs...)
}

func (p *busProducerImpl) Schedule(ctx context.Context, at time.Time, msgs ...StructuredMessage) error {
	return p.scheduleImpl(ctx, &at, msgs...)
}

func (p *busProducerImpl) createFromBaseConfig() BusProducerConfig {
	config := BusProducerConfig{
		Middlewares:      make([]BusProducerMiddleware, 0, len(p.baseConfig.Middlewares)),
		MetadataBuilders: make([]MetadataBuilder, 0, len(p.baseConfig.MetadataBuilders)),
	}

	config.Middlewares = append(config.Middlewares, p.baseConfig.Middlewares...)
	config.MetadataBuilders = append(config.MetadataBuilders, p.baseConfig.MetadataBuilders...)
	return config
}

func (p *busProducerImpl) registerImpl(topic Topic, msg RegisterMessageFunc, config BusProducerConfig) error {
	schema, keyBuilder := msg()
	msgType := schema.Type()
	if msgType == "" {
		return fmt.Errorf("blank message %T must return const value of type", schema)
	}

	if existedTypes, ok := p.topicMessages[topic]; ok {
		if _, ok := existedTypes[msgType]; ok {
			return fmt.Errorf("message type %s already registered in topic", msgType)
		}
	} else {
		p.topicMessages[topic] = make(map[string]struct{})
	}

	msgReflectType := reflect.TypeOf(schema)
	p.producers[producerKey{Topic: topic, Message: msgReflectType}] = p.buildProducer(config, keyBuilder)
	p.messageTopics[msgReflectType] = append(p.messageTopics[msgReflectType], topic)
	p.topicMessages[topic][msgType] = struct{}{}

	return nil
}

func (p *busProducerImpl) buildProducer(config BusProducerConfig, keyBuilder KeyBuilder) BusProduce {
	metadataBuilders := config.MetadataBuilders
	serializePayloadImpl := func(ctx context.Context, msg StructuredMessage) ([]byte, error) {
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

		payload, err := p.serializer.Serialize(msg, meta)
		if err != nil {
			return nil, fmt.Errorf("serialize message %T: %w", msg, err)
		}

		return payload, nil
	}

	if keyBuilder == nil {
		keyBuilder = func(StructuredMessage) string { return "" }
	}

	produce := func(ctx context.Context, topic Topic, msg StructuredMessage, at *time.Time) error {
		payload, err := serializePayloadImpl(ctx, msg)
		if err != nil {
			return fmt.Errorf("serialize message %T: %w", msg, err)
		}

		rawMsg := &Message{
			ID:      msg.ID(),
			Topic:   topic,
			Key:     keyBuilder(msg),
			Payload: payload,
		}

		err = p.producerImpl(ctx, rawMsg, at)
		if err != nil {
			return fmt.Errorf("produce message: %w", err)
		}

		return nil
	}
	for i := len(config.Middlewares) - 1; i >= 0; i-- {
		produce = config.Middlewares[i](produce)
	}

	return produce
}

func (p *busProducerImpl) scheduleImpl(ctx context.Context, at *time.Time, msgs ...StructuredMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	messageTopics := make(map[reflect.Type][]Topic, len(msgs))
	messageProducers := make(map[producerKey]BusProduce, len(messageTopics))
	for _, msg := range msgs {
		msgType := reflect.TypeOf(msg)

		topics, ok := p.messageTopics[msgType]
		if !ok {
			return fmt.Errorf("unknown message type %T", msg)
		}

		messageTopics[msgType] = append(messageTopics[msgType], topics...)

		for _, topic := range topics {
			key := producerKey{topic, msgType}
			producer, ok := p.producers[key]
			if !ok {
				return fmt.Errorf("unknown message type %T", msg)
			}

			messageProducers[key] = producer
		}
	}

	for _, msg := range msgs {
		msgType := reflect.TypeOf(msg)
		topics := messageTopics[msgType]

		for _, topic := range topics {
			producerImpl := p.producers[producerKey{topic, msgType}]
			err := producerImpl(ctx, topic, msg, at)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func WithBusProducerObservability(observer observability.Observer, fields ...observability.Field) BusProducerOption {
	if len(fields) == 0 {
		return func(*BusProducerConfig) {}
	}

	metadataBuilder := func(ctx context.Context) (Metadata, error) { //nolint:unparam
		values := make(map[observability.Field]string, len(fields))
		for _, field := range fields {
			value := observer.Field(ctx, field)
			if value != "" {
				values[field] = value
			}
		}
		if len(values) == 0 {
			return nil, nil
		}

		data := make(Metadata, len(values))
		for key, value := range values {
			data[fmt.Sprintf("%s%s", observabilityMetaKeyPrefix, key)] = value
		}

		return data, nil
	}

	return func(config *BusProducerConfig) {
		config.MetadataBuilders = append(config.MetadataBuilders, metadataBuilder)
	}
}

func WithBusProducerMetrics(metrics metric.Metrics) BusProducerOption {
	mw := func(impl BusProduce) BusProduce {
		return func(ctx context.Context, topic Topic, msg StructuredMessage, scheduleAt *time.Time) error {
			err := impl(ctx, topic, msg, scheduleAt)

			metrics.With(metric.Labels{
				"topic":   topic,
				"type":    msg.Type(),
				"success": err == nil,
			}).Increment("msg_produce_attempts_total")
			return err
		}
	}

	return func(config *BusProducerConfig) {
		config.Middlewares = append(config.Middlewares, mw)
	}
}

func WithBusProducerLogging(logger log.Logger, infoLevel, errorLevel log.Level) BusProducerOption {
	mw := func(impl BusProduce) BusProduce {
		return func(ctx context.Context, topic Topic, msg StructuredMessage, scheduleAt *time.Time) error {
			loggerWithFields := logger.With(log.Fields{
				"topic":       topic,
				"messageID":   msg.ID(),
				"messageType": msg.Type(),
				"scheduleAt":  scheduleAt,
			})

			err := impl(ctx, topic, msg, scheduleAt)
			if err != nil {
				loggerWithFields.
					WithError(err).
					Log(ctx, errorLevel, "message producing failed")
			} else {
				loggerWithFields.Log(ctx, infoLevel, "message successfully produced")
			}

			return err
		}
	}

	return func(config *BusProducerConfig) {
		config.Middlewares = append(config.Middlewares, mw)
	}
}
