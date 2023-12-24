package cmd

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/klwxsrx/go-service-template/pkg/log"
)

func HandleAppPanic(ctx context.Context, logger log.Logger) (panicCaught bool) {
	msg := recover()
	if msg == nil {
		return false
	}

	logger.WithField("panic", log.Fields{
		"message": fmt.Sprintf("%v", msg),
		"stack":   string(debug.Stack()),
	}).Error(ctx, "app failed with panic")
	return true
}
