package message

import (
	"github.com/google/uuid"
)

type Message struct {
	ID      uuid.UUID
	Topic   string
	Key     string
	Payload []byte
}
