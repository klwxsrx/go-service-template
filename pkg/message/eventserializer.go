package message

import (
	"errors"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

var (
	ErrEventTypeDecoderEventTypeNotFound = errors.New("event type not found")
)

type EventTypeDecoder interface {
	EventType(msg *Message) (string, error)
}

var (
	ErrEventSerializerUnknownEventType = errors.New("unknown event type")
)

type EventSerializer interface {
	Serialize(e event.Event) (*Message, error)
}

type EventSerializerTyped[T event.Event] interface {
	Serialize(e T) (*Message, error)
	Deserialize(msg *Message) (T, error)
}
