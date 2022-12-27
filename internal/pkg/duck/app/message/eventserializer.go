package message

import (
	"fmt"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

type eventSerializer struct{}

func (s *eventSerializer) Serialize(e event.Event) (*message.Message, error) {
	switch T := e.(type) {
	case domain.EventDuckCreated:
		return EventSerializerDuckCreated.Serialize(T)
	default:
		return nil, fmt.Errorf("invalid event %v, %s expected", e, e.Type())
	}
}

func NewEventSerializer() message.EventSerializer {
	return &eventSerializer{}
}
