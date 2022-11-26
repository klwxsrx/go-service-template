package metric

import "time"

type Metrics interface {
	Increment(key string, keyValueTags ...string)
	Duration(key string, duration time.Duration, keyValueTags ...string)
}
