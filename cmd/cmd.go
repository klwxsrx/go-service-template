package cmd

import (
	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
)

const gooseServiceDestinationName = "goose"

func MustInitGooseHTTPClient(observer observability.Observer, metrics metric.Metrics, logger log.Logger) http.Client {
	gooseServiceHost := env.Must(env.ParseString("GOOSE_SERVICE_URL"))
	return getHTTPClient(
		gooseServiceHost,
		gooseServiceDestinationName,
		observer,
		metrics,
		logger,
	)
}

func getHTTPClient(
	baseURL string,
	destinationName string,
	observer observability.Observer,
	metrics metric.Metrics,
	logger log.Logger,
) http.Client {
	return http.NewClient(
		http.WithBaseURL(baseURL),
		http.WithRequestObservability(observer, http.DefaultRequestIDHeader),
		http.WithRequestLogging(destinationName, logger, log.LevelInfo, log.LevelWarn),
		http.WithRequestMetrics(destinationName, metrics),
	)
}
