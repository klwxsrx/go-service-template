package env

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"os"
	"strconv"
)

func MustGetString(ctx context.Context, key string, logger log.Logger) string {
	str, ok := LookupString(key)
	if !ok {
		logger.Fatalf(ctx, "failed to lookup string env %s, not found or invalid", key)
	}
	return str
}

func MustGetInt(ctx context.Context, key string, logger log.Logger) int {
	i, ok := LookupInt(key)
	if !ok {
		logger.Fatalf(ctx, "failed to lookup int env %s, not found or invalid", key)
	}
	return i
}

func MustGetBool(ctx context.Context, key string, logger log.Logger) bool {
	b, ok := LookupBool(key)
	if !ok {
		logger.Fatalf(ctx, "failed to lookup bool env %s, not found or invalid", key)
	}
	return b
}

func LookupString(key string) (string, bool) {
	return os.LookupEnv(key)
}

func LookupInt(key string) (int, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return i, true
}

func LookupBool(key string) (bool, bool) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return false, false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, false
	}
	return b, true
}
