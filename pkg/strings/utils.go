package strings

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type (
	SupportedValueParsingTypes interface {
		bool | int | uint | float64 | string | time.Time | time.Duration | uuid.UUID
	}

	SupportedPointerParsingTypes interface {
		*bool | *int | *uint | *float64 | *string | *time.Time | *time.Duration | *uuid.UUID
	}
)

func ParseTypedValue[T SupportedValueParsingTypes | SupportedPointerParsingTypes](value string) (T, error) {
	var v any
	var err error
	var blank T
	switch any(blank).(type) {
	case bool:
		v, err = strconv.ParseBool(value)
	case *bool:
		v, err = wrapWithPointer(strconv.ParseBool(value))
	case int:
		v, err = strconv.Atoi(value)
	case *int:
		v, err = wrapWithPointer(strconv.Atoi(value))
	case uint:
		v, err = strconv.ParseUint(value, 10, 64)
	case *uint:
		v, err = wrapWithPointer(strconv.ParseUint(value, 10, 64))
	case float64:
		v, err = strconv.ParseFloat(value, 64)
	case *float64:
		v, err = wrapWithPointer(strconv.ParseFloat(value, 64))
	case string:
		v, err = value, nil
	case *string:
		v, err = &value, nil
	case time.Time:
		v, err = parseTimeImpl(value)
	case *time.Time:
		v, err = wrapWithPointer(parseTimeImpl(value))
	case time.Duration:
		v, err = time.ParseDuration(value)
	case *time.Duration:
		v, err = wrapWithPointer(time.ParseDuration(value))
	case uuid.UUID:
		v, err = uuid.Parse(value)
	case *uuid.UUID:
		v, err = wrapWithPointer(uuid.Parse(value))
	default:
		return blank, fmt.Errorf("unsupported value type %T", blank)
	}
	if err != nil {
		return blank, fmt.Errorf("convert to type %T: %w", blank, err)
	}

	return v.(T), nil
}

func parseTimeImpl(value string) (any, error) {
	const timeTimeParseError = "possible formats RFC3339 or RFC3339Nano or UnixTime"

	var v any
	var err error
	v, err = time.Parse(time.RFC3339, value)
	if err == nil {
		return v, nil
	}

	v, err = time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return v, nil
	}

	var unixTime int64
	unixTime, err = strconv.ParseInt(value, 10, 32)
	if err != nil {
		return v, errors.New(timeTimeParseError)
	}
	if unixTime < 0 {
		err = errors.New(timeTimeParseError)
		return v, err
	}
	v = time.Unix(unixTime, 0)

	return v, nil
}

func wrapWithPointer[T any](v T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}

	return &v, nil
}
