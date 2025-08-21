package message

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/strings"
)

type Subscriber string

func NewSubscriberService(name string) Subscriber {
	return NewSubscriberCustom(name, "service")
}

func NewSubscriberCustom(name, custom string) Subscriber {
	return NewSubscriber(fmt.Sprintf("%s-%s", name, custom))
}

func NewSubscriber(name string) Subscriber {
	name = strings.ToKebabCase(name)
	return Subscriber(name)
}
