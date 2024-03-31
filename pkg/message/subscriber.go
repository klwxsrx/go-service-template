package message

import (
	"fmt"

	"github.com/iancoleman/strcase"
)

type SubscriberName string

func NewSubscriberName(name string) SubscriberName {
	name = strcase.ToKebab(name)
	return SubscriberName(name)
}

func NewSubscriberServiceName(name string) SubscriberName {
	return NewSubscriberName(fmt.Sprintf("%s-service", name))
}

func NewSubscriberCustomName(name, custom string) SubscriberName {
	return NewSubscriberName(fmt.Sprintf("%s-%s", name, custom))
}
