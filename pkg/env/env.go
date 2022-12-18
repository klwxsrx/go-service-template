package env

import (
	"fmt"
	"os"
	"time"
)

func Must[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Errorf("failed to parse environment: %w", err))
	}
	return val
}

func ParseString(key string) (string, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return "", notFoundError(key, "string")
	}
	return str, nil
}

func ParseDuration(key string) (time.Duration, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return 0, notFoundError(key, "duration")
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return 0, invalidValueError(key, "duration")
	}
	return d, nil
}

func notFoundError(key, varType string) error {
	return fmt.Errorf("env \"%s\" with type \"%s\" not found", key, varType)
}

func invalidValueError(key, varType string) error {
	return fmt.Errorf("env \"%s\" with type \"%s\" has invalid value", key, varType)
}
