//go:generate mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Metrics=Metrics"
package metric

import "time"

type Labels map[string]any

type Metrics interface {
	With(labels Labels) Metrics
	WithLabel(name string, value any) Metrics
	Increment(metric string)
	Count(metric string, increase int)
	Gauge(metric string, current int)
	Duration(metric string, duration time.Duration)
}
