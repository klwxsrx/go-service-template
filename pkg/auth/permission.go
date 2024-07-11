package auth

import (
	"context"
	"errors"
	"fmt"
)

var ErrPermissionDenied = errors.New("permission denied")

type (
	PermissionService[T Principal] interface {
		Check(context.Context, Permission[T]) error
	}

	Permission[T Principal] func(Authentication[T]) (bool, error)

	permissionService[T Principal] struct{}
)

func NewPermissionService[T Principal]() PermissionService[T] {
	return permissionService[T]{}
}

func (p permissionService[T]) Check(ctx context.Context, permission Permission[T]) error {
	auth, ok := GetAuthentication[T](ctx)
	if !ok {
		return errors.New("authentication not found")
	}

	allowed, err := permission(auth)
	if err != nil {
		return fmt.Errorf("check permission: %w", err)
	}
	if !allowed {
		return ErrPermissionDenied
	}

	return nil
}
