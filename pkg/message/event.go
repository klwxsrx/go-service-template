package message

import (
	"errors"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

var (
	ErrEventSerializerMessageIsNotValidEvent = errors.New("message is not valid event")
	ErrEventSerializerUnknownEventType       = errors.New("unknown event type")
)

type EventSerializer interface {
	Serialize(e event.Event) (*Message, error)
	Deserialize(msg *Message) (event.Event, error)
}
