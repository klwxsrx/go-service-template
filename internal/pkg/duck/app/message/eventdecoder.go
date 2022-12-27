package message

import (
	"encoding/json"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

type eventTypeDecoder struct{}

func (d *eventTypeDecoder) EventType(msg *message.Message) (string, error) {
	var base baseMessagePayload
	err := json.Unmarshal(msg.Payload, &base)
	if err != nil || base.EventType == "" {
		return "", fmt.Errorf("%w %v", message.ErrEventTypeDecoderEventTypeNotFound, msg.ID)
	}

	return base.EventType, nil
}

func NewEventTypeDecoder() message.EventTypeDecoder {
	return &eventTypeDecoder{}
}
