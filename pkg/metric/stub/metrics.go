package stub

import (
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"time"
)

type metrics struct{}

func (m metrics) Increment(_ string, _ ...string) {}

func (m metrics) Duration(_ string, _ time.Duration, _ ...string) {}

func NewMetrics() metric.Metrics {
	return metrics{}
}
