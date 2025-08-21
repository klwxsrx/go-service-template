package metric

import (
	"time"
)

type metricsStub struct{}

func NewMetricsStub() Metrics {
	return metricsStub{}
}

func (s metricsStub) With(Labels) Metrics {
	return s
}

func (s metricsStub) WithLabel(string, any) Metrics {
	return s
}

func (s metricsStub) Increment(string) {}

func (s metricsStub) Count(string, int) {}

func (s metricsStub) Gauge(string, int) {}

func (s metricsStub) Duration(string, time.Duration) {}
