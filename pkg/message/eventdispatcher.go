package message

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type eventDispatcher struct {
	producer   Producer
	serializer EventSerializer
}

func (d *eventDispatcher) Dispatch(ctx context.Context, events []event.Event) error {
	msgs := make([]Message, 0, len(events))

	for _, e := range events {
		msg, err := d.serializer.Serialize(e)
		if err != nil {
			return fmt.Errorf("failed to serialize event to message: %w", err)
		}
		msgs = append(msgs, *msg)
	}

	for _, msg := range msgs {
		v := msg
		err := d.producer.Send(ctx, &v)
		if err != nil {
			return fmt.Errorf("failed to send event message: %w", err)
		}
	}

	return nil
}

func NewEventDispatcher(
	producer Producer,
	serializer EventSerializer,
) event.Dispatcher {
	return &eventDispatcher{
		producer:   producer,
		serializer: serializer,
	}
}
