package message

import (
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

var SubscriberName = message.NewSubscriberServiceName(domain.Name)
