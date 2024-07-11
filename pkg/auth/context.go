package auth

import (
	"context"
	"errors"
)

const authenticationContextKey contextKey = iota

type contextKey int

func WithAuthentication[T Principal](ctx context.Context, auth Authentication[T]) context.Context {
	var principal *Principal
	if auth.Principal() != nil {
		p := Principal(*auth.Principal())
		principal = &p
	}

	return context.WithValue(ctx, authenticationContextKey, Auth[Principal]{principal})
}

func GetAuthentication[T Principal](ctx context.Context) (Authentication[T], bool) {
	authentication, ok := ctx.Value(authenticationContextKey).(Authentication[Principal])
	if !ok {
		return nil, false
	}

	var principal *T
	if authentication.Principal() != nil {
		p, ok := (*authentication.Principal()).(T)
		if !ok {
			return nil, false
		}
		principal = &p
	}

	return Auth[T]{principal}, ok
}

func IsAuthenticated(ctx context.Context) (bool, error) {
	result, ok := ctx.Value(authenticationContextKey).(Authentication[Principal])
	if !ok {
		return false, errors.New("authentication not found")
	}

	return result.IsAuthenticated(), nil
}
