package env

import (
	"fmt"
	"os"
	"strings"

	pkgstrings "github.com/klwxsrx/go-service-template/pkg/strings"
)

type (
	supportedTypes interface {
		pkgstrings.SupportedValueParsingTypes
	}

	supportedOptionalTypes interface {
		pkgstrings.SupportedPointerParsingTypes
	}
)

func Parse[T supportedTypes](key string) (result T, err error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return result, notFoundError(key, result)
	}

	result, err = pkgstrings.ParseTypedValue[T](str)
	if err != nil {
		return result, invalidValueError(key, result, err)
	}

	return result, nil
}

func ParseOptional[T supportedOptionalTypes](key string) (result T, err error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil
	}

	result, err = pkgstrings.ParseTypedValue[T](str)
	if err != nil {
		return result, invalidValueError(key, result, err)
	}

	return result, nil
}

func ParseList[T supportedTypes](key, delimiter string) ([]T, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		var empty []T
		return nil, notFoundError(key, empty)
	}

	result, err := parseListImpl[T](key, str, delimiter)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		var empty []T
		return nil, fmt.Errorf("env %s with type %T is empty", strings.ToUpper(key), empty)
	}

	return result, nil
}

func ParseListOptional[T supportedTypes](key, delimiter string) ([]T, error) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil
	}

	return parseListImpl[T](key, str, delimiter)
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(fmt.Errorf("parse environment: %w", err))
	}
	return val
}

func parseListImpl[T supportedTypes](key, value, delimiter string) ([]T, error) {
	valueList := strings.Split(value, delimiter)
	resultList := make([]T, 0, len(valueList))
	for _, value := range valueList {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		t, err := pkgstrings.ParseTypedValue[T](value)
		if err != nil {
			var empty []T
			return nil, invalidValueError(key, empty, err)
		}

		resultList = append(resultList, t)
	}

	return resultList, nil
}

func notFoundError(key string, varType any) error {
	return fmt.Errorf("env %s with type %T not found", strings.ToUpper(key), varType)
}

func invalidValueError(key string, varType any, err error) error {
	return fmt.Errorf("env %s with type %T is invalid: %w", strings.ToUpper(key), varType, err)
}
