package message

import (
	"errors"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

var (
	ErrEventSerializeUnknownEventType   = errors.New("unknown event type")
	ErrEventDeserializeUnknownEventType = errors.New("unknown event type")
	ErrEventDeserializeNotValidEvent    = errors.New("message is not valid event")
)

type EventSerializer interface {
	Serialize(evt event.Event) (*Message, error)
	EventDeserializer
}

type EventDeserializer interface {
	ParseType(msg *Message) (string, error)
	Deserialize(msg *Message) (event.Event, error)
}
