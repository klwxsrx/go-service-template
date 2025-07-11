package message

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/strings"
)

type SubscriberName string

func NewSubscriberName(name string) SubscriberName {
	name = strings.ToKebabCase(name)
	return SubscriberName(name)
}

func NewSubscriberServiceName(name string) SubscriberName {
	return NewSubscriberName(fmt.Sprintf("%s-service", name))
}

func NewSubscriberCustomName(name, custom string) SubscriberName {
	return NewSubscriberName(fmt.Sprintf("%s-%s", name, custom))
}
