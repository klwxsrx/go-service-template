package http

import (
	"fmt"

	"github.com/iancoleman/strcase"

	pkgenv "github.com/klwxsrx/go-service-template/pkg/env"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	pkglog "github.com/klwxsrx/go-service-template/pkg/log"
	pkgmetric "github.com/klwxsrx/go-service-template/pkg/metric"
	pkgobservability "github.com/klwxsrx/go-service-template/pkg/observability"
)

type Destination string

const (
	DestinationGooseService Destination = "goose"
)

type ClientFactory struct {
	observer pkgobservability.Observer
	metrics  pkgmetric.Metrics
	logger   pkglog.Logger
}

func NewClientFactory(
	observer pkgobservability.Observer,
	metrics pkgmetric.Metrics,
	logger pkglog.Logger,
) *ClientFactory {
	return &ClientFactory{
		observer: observer,
		metrics:  metrics,
		logger:   logger,
	}
}

func (f *ClientFactory) MustInitClient(dest Destination, extraOpts ...pkghttp.ClientOption) pkghttp.Client {
	hostEnv := fmt.Sprintf("%s_SERVICE_URL", strcase.ToScreamingSnake(string(dest)))
	host := pkgenv.Must(pkgenv.ParseString(hostEnv))
	return f.httpClient(host, string(dest), extraOpts...)
}

func (f *ClientFactory) httpClient(
	baseURL string,
	destinationName string,
	extraOpts ...pkghttp.ClientOption,
) pkghttp.Client {
	opts := append([]pkghttp.ClientOption{
		pkghttp.WithClientBaseURL(baseURL),
		pkghttp.WithRequestObservability(f.observer, pkghttp.DefaultRequestIDHeader),
		pkghttp.WithRequestLogging(destinationName, f.logger, pkglog.LevelInfo, pkglog.LevelWarn),
		pkghttp.WithRequestMetrics(destinationName, f.metrics),
	}, extraOpts...)

	return pkghttp.NewClient(opts...)
}
