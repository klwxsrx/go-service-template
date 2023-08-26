package strings

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func ParseTypedValue[T any](value string) (T, error) {
	var v any
	var err error
	var blank T
	switch any(blank).(type) {
	case bool:
		v, err = strconv.ParseBool(value)
	case int:
		v, err = strconv.Atoi(value)
	case uint:
		v, err = strconv.ParseUint(value, 10, 64)
	case float64:
		v, err = strconv.ParseFloat(value, 64)
	case string:
		v, err = value, nil
	case time.Time:
		v, err = time.Parse(time.RFC3339, value)
		if err != nil {
			break
		}
		v, err = time.Parse(time.RFC3339Nano, value)
		if err != nil {
			break
		}
		var unixTime int64
		unixTime, err = strconv.ParseInt(value, 10, 32)
		if err != nil {
			break
		}
		if unixTime < 0 {
			err = errors.New("got negative seconds value")
			break
		}

		v = time.Unix(unixTime, 0)
	case time.Duration:
		v, err = time.ParseDuration(value)
	case uuid.UUID:
		v, err = uuid.Parse(value)
	default:
		return blank, fmt.Errorf("unsupported value type %T", blank)
	}

	if err != nil {
		return blank, fmt.Errorf("failed to convert to type %T: %w", blank, err)
	}
	return v.(T), nil
}
