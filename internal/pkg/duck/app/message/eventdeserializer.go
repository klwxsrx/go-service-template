package message

import (
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

func NewEventDeserializer() message.EventDeserializer {
	return message.NewJSONEventDeserializer(
		message.RegisterJSONEvent[domain.EventDuckCreated](domain.EventTypeDuckCreated),
	)
}
