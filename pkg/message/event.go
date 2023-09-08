package message

import (
	"context"
	"errors"
	"fmt"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/event"
)

const messageClassEvent = "domainEvent"

type eventDispatcher struct {
	bus Bus
}

func NewEventDispatcher(
	bus Bus,
) event.Dispatcher {
	return eventDispatcher{
		bus: bus,
	}
}

func (d eventDispatcher) Dispatch(ctx context.Context, events []event.Event) error {
	for _, evt := range events {
		err := d.bus.Produce(ctx, messageClassEvent, evt)
		if err != nil {
			return fmt.Errorf("failed to dispatch event: %w", err)
		}
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
				fmt.Errorf("failed to get aggregate name for %T: blank event must return const value", blank)
		}

		eventType := blank.Type()
		if eventType == "" {
			return "",
				"",
				nil,
				nil,
				fmt.Errorf("failed to get event type for %T: blank event must return const value", blank)
		}

		return messageClassEvent,
			eventType,
			func(domainName string) string {
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

func RegisterEventHandler[T event.Event](handler event.TypedHandler[T]) RegisterHandlerFunc {
	return func(publisherDomain string, deserializer Deserializer) (string, ConsumptionType, Handler, error) {
		var blank T
		aggregateName := blank.AggregateName()
		if aggregateName == "" {
			return "",
				"",
				nil,
				fmt.Errorf("failed to get aggregate name for %T: blank event must return const value", blank)
		}

		eventType := blank.Type()
		if eventType == "" {
			return "",
				"",
				nil,
				fmt.Errorf("failed to get event type for %T: blank event must return const value", blank)
		}

		err := deserializer.RegisterDeserializer(publisherDomain, messageClassEvent, eventType, TypedDeserializer[T]())
		if err != nil {
			return "",
				"",
				nil,
				fmt.Errorf("failed to register event %T deserializer: %w", blank, err)
		}

		return buildEventTopic(publisherDomain, aggregateName),
			ConsumptionTypeSingle,
			eventHandlerImpl[T](publisherDomain, messageClassEvent, handler, deserializer),
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
	messageClass string,
	handler event.TypedHandler[T],
	deserializer Deserializer,
) Handler {
	return func(ctx context.Context, msg *Message) error {
		evt, err := deserializer.Deserialize(ctx, publisherDomain, messageClass, msg)
		if errors.Is(err, ErrDeserializeNotValidMessage) || errors.Is(err, ErrDeserializeUnknownMessage) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to deserialize message %v: %w", msg.ID, err)
		}

		concreteEvent, ok := evt.(T)
		if !ok {
			return fmt.Errorf("invalid event struct type %T for messageID %v, expected %T", evt, msg.ID, concreteEvent)
		}

		return handler(ctx, concreteEvent)
	}
}
