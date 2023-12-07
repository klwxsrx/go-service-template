package stub

import (
	"time"

	"github.com/klwxsrx/go-service-template/pkg/metric"
)

type metrics struct{}

func NewMetrics() metric.Metrics {
	return metrics{}
}

func (m metrics) With(_ metric.Labels) metric.Metrics {
	return m
}

func (m metrics) WithLabel(_ string, _ any) metric.Metrics {
	return m
}

func (m metrics) Increment(_ string) {}

func (m metrics) Count(_ string, _ int) {}

func (m metrics) Gauge(_ string, _ int) {}

func (m metrics) Duration(_ string, _ time.Duration) {}
