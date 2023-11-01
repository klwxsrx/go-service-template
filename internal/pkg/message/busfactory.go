package message

import (
	pkglog "github.com/klwxsrx/go-service-template/pkg/log"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgmetric "github.com/klwxsrx/go-service-template/pkg/metric"
	pkgobservability "github.com/klwxsrx/go-service-template/pkg/observability"
)

type BusFactory struct {
	observer pkgobservability.Observer
	metrics  pkgmetric.Metrics
	logger   pkglog.Logger
}

func (f *BusFactory) New(domainName string, storage pkgmessage.OutboxStorage) pkgmessage.Bus {
	return pkgmessage.NewBus(
		domainName,
		storage,
		pkgmessage.WithObservability(f.observer),
		pkgmessage.WithMetrics(f.metrics),
		pkgmessage.WithLogging(f.logger, pkglog.LevelInfo, pkglog.LevelWarn),
	)
}

func NewBusFactory(
	observer pkgobservability.Observer,
	metrics pkgmetric.Metrics,
	logger pkglog.Logger,
) *BusFactory {
	return &BusFactory{
		observer: observer,
		metrics:  metrics,
		logger:   logger,
	}
}
