package metric

import (
	"time"
)

type metricsStub struct{}

func NewMetricsStub() Metrics {
	return metricsStub{}
}

func (s metricsStub) With(_ Labels) Metrics {
	return s
}

func (s metricsStub) WithLabel(_ string, _ any) Metrics {
	return s
}

func (s metricsStub) Increment(_ string) {}

func (s metricsStub) Count(_ string, _ int) {}

func (s metricsStub) Gauge(_ string, _ int) {}

func (s metricsStub) Duration(_ string, _ time.Duration) {}
