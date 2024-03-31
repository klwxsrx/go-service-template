package goose

import (
	"github.com/klwxsrx/go-service-template/internal/duck/app/goose"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
)

const domainName = "goose"

var DomainEventTopicDefinitionGoose = pkgmessage.NewDomainEventTopicSubscription(domainName, goose.AggregateNameGoose)
