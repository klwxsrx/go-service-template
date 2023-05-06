package message

import (
	"context"
	"errors"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

func NewEventHandler[T event.Event](domainName string, handler event.TypedHandler[T]) (Handler, error) {
	deserializer := newJSONEventDeserializer()
	err := deserializer.RegisterJSONEvent(domainName, registerDeserializerTyped[T]())
	if err != nil {
		return nil, fmt.Errorf("failed to register event deserializer: %w", err)
	}
	return eventHandlerImpl[T](domainName, handler, deserializer), nil
}

func eventHandlerImpl[T event.Event](
	domainName string,
	handler event.TypedHandler[T],
	deserializer *jsonEventDeserializer,
) Handler {
	return func(ctx context.Context, msg *Message) error {
		evt, err := deserializer.Deserialize(domainName, msg)
		if errors.Is(err, errEventDeserializeNotValidEvent) || errors.Is(err, errEventDeserializeUnknownEvent) {
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
