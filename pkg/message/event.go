package message

import (
	"context"
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/event"
)

type (
	EventDispatcher interface {
		event.Dispatcher
		Registry
	}

	eventDispatcher struct {
		bus BusProducer
	}
)

func NewEventDispatcher(bus BusProducer) EventDispatcher {
	return eventDispatcher{
		bus: bus,
	}
}

func (d eventDispatcher) Dispatch(ctx context.Context, events ...event.Event) error {
	if len(events) == 0 {
		return nil
	}

	msgs := make([]StructuredMessage, 0, len(events))
	for _, evt := range events {
		msgs = append(msgs, StructuredMessage(evt))
	}

	err := d.bus.Produce(ctx, msgs...)
	if err != nil {
		return fmt.Errorf("dispatch event: %w", err)
	}

	return nil
}

func (d eventDispatcher) Register(messages TopicMessages, opts ...BusProducerOption) error {
	return d.bus.Register(messages, opts...)
}

func RegisterEvent[T event.Event]() RegisterMessageFunc {
	return func() (StructuredMessage, KeyBuilder) {
		keyBuilder := func(msg StructuredMessage) string {
			evt, ok := msg.(T)
			if !ok {
				return ""
			}

			return evt.AggregateID().String()
		}

		var blank T
		return blank, keyBuilder
	}
}

func RegisterEventHandlers[T event.Event](handlers ...event.TypedHandler[T]) RegisterHandlersFunc {
	return func() (StructuredMessage, PayloadDeserializer, []TypedHandler[StructuredMessage]) {
		handlersImpl := make([]TypedHandler[StructuredMessage], 0, len(handlers))
		for _, handler := range handlers {
			handlersImpl = append(handlersImpl, func(ctx context.Context, msg StructuredMessage) error {
				evt, ok := msg.(T)
				if !ok {
					return fmt.Errorf("invalid event struct type %T for messageID %v, expected %T", msg, msg.ID(), evt)
				}

				return handler(ctx, evt)
			})
		}

		var blank T
		return blank, PayloadDeserializerImpl[T], handlersImpl
	}
}

func NewTopicDomainEvent(domainName, aggregateName string, customTags ...string) Topic {
	return NewTopic(
		"domain-event",
		WithTopicDomainName(domainName),
		WithTopicAggregateName(aggregateName),
		WithTopicCustomTags(customTags...),
	)
}
