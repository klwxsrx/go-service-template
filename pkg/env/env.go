package env

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	pkgstrings "github.com/klwxsrx/go-service-template/pkg/strings"
)

type availableTypes interface {
	bool | int | float64 | string | time.Time | time.Duration | uuid.UUID
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Errorf("failed to parse environment: %w", err))
	}
	return val
}

func ParseBool(key string) (bool, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return false, notFoundError(key, "boolean")
	}
	b, err := pkgstrings.ParseTypedValue[bool](str)
	if err != nil {
		return false, fmt.Errorf("%w, true\\felse or 1\\0 expected", invalidValueError(key, "boolean"))
	}
	return b, nil
}

func ParseInt(key string) (int, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return 0, notFoundError(key, "integer")
	}
	i, err := pkgstrings.ParseTypedValue[int](str)
	if err != nil {
		return 0, invalidValueError(key, "integer")
	}
	return i, nil
}

func ParseUint(key string) (uint, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return 0, notFoundError(key, "unsigned integer")
	}
	i, err := pkgstrings.ParseTypedValue[uint](str)
	if err != nil {
		return 0, invalidValueError(key, "unsigned integer")
	}
	return i, nil
}

func ParseFloat(key string) (float64, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return 0, notFoundError(key, "float")
	}
	f, err := pkgstrings.ParseTypedValue[float64](str)
	if err != nil {
		return 0, invalidValueError(key, "float")
	}
	return f, nil
}

func ParseString(key string) (string, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return "", notFoundError(key, "string")
	}
	return str, nil
}

func ParseTime(key string) (time.Time, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return time.Time{}, notFoundError(key, "time")
	}
	t, err := pkgstrings.ParseTypedValue[time.Time](str)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w, RFC3339, RFC3339Nano or Unix time expected", invalidValueError(key, "time"))
	}
	return t, nil
}

func ParseDuration(key string) (time.Duration, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return 0, notFoundError(key, "duration")
	}
	d, err := pkgstrings.ParseTypedValue[time.Duration](str)
	if err != nil {
		return 0, invalidValueError(key, "duration")
	}
	return d, nil
}

func ParseUUID(key string) (uuid.UUID, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return uuid.UUID{}, notFoundError(key, "uuid")
	}
	id, err := pkgstrings.ParseTypedValue[uuid.UUID](str)
	if err != nil {
		return uuid.UUID{}, invalidValueError(key, "uuid")
	}
	return id, nil
}

func ParseList[T availableTypes](key string, delimiter string) ([]T, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return nil, notFoundError(key, "list")
	}

	strList := strings.Split(str, delimiter)
	resultList := make([]T, 0, len(strList))
	for _, str := range strList {
		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}
		t, err := pkgstrings.ParseTypedValue[T](str)
		if err != nil {
			return nil, invalidValueError(key, "list")
		}
		resultList = append(resultList, t)
	}

	return resultList, nil
}

func notFoundError(key, varType string) error {
	return fmt.Errorf("env %s with type %s not found", key, varType)
}

func invalidValueError(key, varType string) error {
	return fmt.Errorf("env %s with type %s has invalid value", key, varType)
}
