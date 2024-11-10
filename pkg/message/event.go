package message

import (
	"context"
	"fmt"
	"time"

	"github.com/klwxsrx/go-service-template/pkg/event"
)

type (
	EventDispatcher interface {
		event.Dispatcher
		ProducerRegistry
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

	err := d.bus.Produce(ctx, msgs, time.Now())
	if err != nil {
		return fmt.Errorf("dispatch event: %w", err)
	}

	return nil
}

func (d eventDispatcher) RegisterMessages(messagesMap TopicMessagesMap) error {
	return d.bus.RegisterMessages(messagesMap)
}

func RegisterEvent[T event.Event]() RegisterMessageFunc {
	return func() (StructuredMessage, KeyBuilderFunc) {
		var blank T
		return blank,
			func(msg StructuredMessage) string {
				evt, ok := msg.(T)
				if !ok {
					return ""
				}

				return evt.AggregateID().String()
			}
	}
}

func RegisterEventHandler[T event.Event](handler event.TypedHandler[T]) RegisterHandlerFunc {
	return func() (StructuredMessage, DeserializerFunc, TypedHandler[StructuredMessage]) {
		var blank T
		return blank, TypedJSONDeserializer[T](), func(ctx context.Context, msg StructuredMessage) error {
			evt, ok := msg.(T)
			if !ok {
				return fmt.Errorf("invalid event struct type %T for messageID %v, expected %T", msg, msg.ID(), evt)
			}

			return handler(ctx, evt)
		}
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

func NewTopicSubscriptionDomainEvent(domainName, aggregateName string, customTags ...string) TopicSubscription {
	return TopicSubscription{
		Topic:           NewTopicDomainEvent(domainName, aggregateName, customTags...),
		ConsumptionType: ConsumptionTypeExclusive,
	}
}
