package message

import (
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

var TopicDomainEventUser = message.NewTopicDomainEvent(domain.Name, domain.AggregateNameUser)
