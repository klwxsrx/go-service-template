package message

import (
	"errors"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type (
	HandlerRegistry interface {
		RegisterHandlers(Subscriber, TopicHandlers, ...ListenerOption) error
	}

	BusListener interface {
		HandlerRegistry
		Workers() []worker.ErrorJob
	}

	TopicHandlers map[Topic][]RegisterHandlersFunc

	busListener[S AcknowledgeStrategy] struct {
		consumers    ConsumerProvider[S]
		queue        ListenerQueueBuilder[S]
		deserializer func() Deserializer
		listeners    map[subscriberKey]listenerData[S]
		opts         []ListenerOption
	}

	listenerData[S AcknowledgeStrategy] struct {
		Consumer     Consumer[S]
		Deserializer Deserializer
		Handlers     map[string][]TypedHandler[StructuredMessage]
		ExtraOpts    []ListenerOption
	}

	subscriberKey struct {
		Subscriber Subscriber
		Topic      Topic
	}
)

func NewBusListener[S AcknowledgeStrategy](
	consumers ConsumerProvider[S],
	processingQueue ListenerQueueBuilder[S],
	deserializer func() Deserializer,
	opts ...ListenerOption,
) BusListener {
	return &busListener[S]{
		consumers:    consumers,
		queue:        processingQueue,
		deserializer: deserializer,
		listeners:    make(map[subscriberKey]listenerData[S]),
		opts:         opts,
	}
}

func (b *busListener[S]) RegisterHandlers(subscriber Subscriber, handlers TopicHandlers, opts ...ListenerOption) error {
	for topic, funcs := range handlers {
		if err := b.registerTopicHandlers(subscriber, topic, funcs, opts...); err != nil {
			return fmt.Errorf("register handler for topic %s by %s: %w", topic, subscriber, err)
		}
	}

	return nil
}

func (b *busListener[S]) Workers() []worker.ErrorJob {
	listeners := make([]worker.ErrorJob, 0, len(b.listeners))
	for _, data := range b.listeners {
		listeners = append(listeners, NewListener[S](
			data.Consumer,
			data.Handlers,
			b.queue,
			data.Deserializer,
			append(b.opts, data.ExtraOpts...)...,
		))
	}

	return listeners
}

func (b *busListener[S]) registerTopicHandlers(
	subscriber Subscriber,
	topic Topic,
	funcs []RegisterHandlersFunc,
	opts ...ListenerOption,
) error {
	key := subscriberKey{subscriber, topic}
	if _, ok := b.listeners[key]; ok {
		return errors.New("handlers already registered")
	}

	deserializer := b.deserializer()
	topicMessageTypes := make(map[string]struct{}, len(funcs))
	handlers := make(map[string][]TypedHandler[StructuredMessage], len(funcs))
	for _, fn := range funcs {
		msgSchema, msgDeserializer, msgHandlers := fn()
		msgType := msgSchema.Type()
		if msgType == "" {
			return fmt.Errorf("blank message %T must return message type const value", msgSchema)
		}

		if _, ok := topicMessageTypes[msgType]; ok {
			return fmt.Errorf("message type %s already registered", msgType)
		}
		topicMessageTypes[msgType] = struct{}{}

		err := deserializer.RegisterDeserializer(msgType, msgDeserializer)
		if err != nil {
			return fmt.Errorf("register deserializer for %T: %w", msgSchema, err)
		}

		handlers[msgType] = msgHandlers
	}

	consumer, err := b.consumers.Consumer(topic, subscriber)
	if err != nil {
		return fmt.Errorf("get consumer: %w", err)
	}

	b.listeners[key] = listenerData[S]{
		Consumer:     consumer,
		Deserializer: deserializer,
		Handlers:     handlers,
		ExtraOpts:    opts,
	}

	return nil
}
