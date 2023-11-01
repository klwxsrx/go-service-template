//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Observer=Observer"
package observability

import (
	"context"
)

type contextKey int

const (
	requestIDContextKey contextKey = iota
)

type Observer interface {
	RequestID(context.Context) (string, bool)
	WithRequestID(context.Context, string) context.Context
}

type observer struct{}

func (o observer) RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok || len(requestID) == 0 {
		return "", false
	}
	return requestID, true
}

func (o observer) WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, id)
}

func New() Observer { // TODO: WithLogging
	return observer{}
}
