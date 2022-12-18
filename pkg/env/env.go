package env

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"os"
	"time"
)

func MustParseString(ctx context.Context, key string, logger log.Logger) string {
	s, ok := ParseString(key)
	if !ok {
		handleLookupEnvFatal(ctx, key, "string", logger)
	}
	return s
}

func ParseString(key string) (string, bool) {
	return os.LookupEnv(key)
}

func MustParseDuration(ctx context.Context, key string, logger log.Logger) time.Duration {
	d, ok := ParseDuration(key)
	if !ok {
		handleLookupEnvFatal(ctx, key, "duration", logger)
	}
	return d
}

func ParseDuration(key string) (time.Duration, bool) {
	str, ok := os.LookupEnv(key)
	if !ok {
		return 0, false
	}
	d, err := time.ParseDuration(str)
	if err != nil {
		return 0, false
	}
	return d, true
}

func handleLookupEnvFatal(ctx context.Context, key, envType string, logger log.Logger) {
	logger.WithField("env", key).Fatal(ctx, fmt.Sprintf("env with type \"%s\" not found or invalid", envType))
}
