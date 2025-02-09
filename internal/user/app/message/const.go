package message

import (
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

var TopicDomainEventUser = message.NewTopicSubscriptionDomainEvent(domain.Name, domain.AggregateNameUser)
