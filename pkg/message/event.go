package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/event"
)

const messageClassEvent = "domainEvent"

type eventDispatcher struct {
	domainName string
	bus        BusProducer
}

func NewEventDispatcher(
	domainName string,
	bus BusProducer,
) event.Dispatcher {
	return eventDispatcher{
		domainName: domainName,
		bus:        bus,
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

	err := d.bus.Produce(ctx, d.domainName, messageClassEvent, msgs, time.Now())
	if err != nil {
		return fmt.Errorf("dispatch event: %w", err)
	}

	return nil
}

func RegisterEvent[T event.Event]() RegisterStructuredMessageFunc {
	return func(domainName string) (messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc, err error) {
		var blank T
		aggregateName := blank.AggregateName()
		if aggregateName == "" {
			return "",
				"",
				nil,
				nil,
				fmt.Errorf("get aggregate name for %T: blank event must return const value", blank)
		}

		eventType := blank.Type()
		if eventType == "" {
			return "",
				"",
				nil,
				nil,
				fmt.Errorf("get event type for %T: blank event must return const value", blank)
		}

		return messageClassEvent,
			eventType,
			func(string) string {
				return buildEventTopic(domainName, aggregateName)
			},
			func(msg StructuredMessage) string {
				evt, ok := msg.(T)
				if !ok {
					return ""
				}
				return evt.AggregateID().String()
			},
			nil
	}
}

func RegisterEventHandler[T event.Event](publisherDomain string, handler event.TypedHandler[T]) RegisterHandlerFunc {
	return func(_ string, deserializer Deserializer) (Handler, []ConsumerSubscription, error) {
		var blank T
		aggregateName := blank.AggregateName()
		if aggregateName == "" {
			return nil,
				nil,
				fmt.Errorf("get aggregate name for %T: blank event must return const value", blank)
		}

		eventType := blank.Type()
		if eventType == "" {
			return nil,
				nil,
				fmt.Errorf("get event type for %T: blank event must return const value", blank)
		}

		err := deserializer.RegisterDeserializer(publisherDomain, messageClassEvent, eventType, TypedDeserializer[T]())
		if err != nil {
			return nil,
				nil,
				fmt.Errorf("register event %T deserializer: %w", blank, err)
		}

		return eventHandlerImpl[T](publisherDomain, handler, deserializer),
			[]ConsumerSubscription{{
				Topic:           buildEventTopic(publisherDomain, aggregateName),
				ConsumptionType: ConsumptionTypeSingle,
			}},
			nil
	}
}

func buildEventTopic(domainName, aggregateName string) string {
	domainName = strcase.ToKebab(domainName)
	aggregateName = strcase.ToKebab(aggregateName)
	return fmt.Sprintf("event.%s-domain.%s-aggregate", domainName, aggregateName)
}

func eventHandlerImpl[T event.Event](
	publisherDomain string,
	handler event.TypedHandler[T],
	deserializer Deserializer,
) Handler {
	return func(ctx context.Context, msg *Message) error {
		evt, err := deserializer.Deserialize(publisherDomain, messageClassEvent, msg)
		if errors.Is(err, ErrDeserializeNotValidMessage) || errors.Is(err, ErrDeserializeUnknownMessage) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("deserialize message %v: %w", msg.ID, err)
		}

		concreteEvent, ok := evt.(T)
		if !ok {
			return fmt.Errorf("invalid event struct type %T for messageID %v, expected %T", evt, msg.ID, concreteEvent)
		}

		return handler(ctx, concreteEvent)
	}
}
