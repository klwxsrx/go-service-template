package event

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type duckCreatedHandler struct {
}

func (h *duckCreatedHandler) Handle(_ context.Context, event event.Event) error {
	fmt.Println(event)
	return nil
}

func NewDuckCreatedHandler() event.Handler {
	return &duckCreatedHandler{}
}
