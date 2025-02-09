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
		Filter(context.Context, UntypedPermissionFilter[T]) (any, error)
	}

	Permission[T Principal]                func(Authentication[T]) (bool, error)
	PermissionFilter[T1 any, T2 Principal] func(Authentication[T2]) (T1, error)
	UntypedPermissionFilter[T Principal]   PermissionFilter[any, T]

	permissionService[T Principal] struct{}
)

func FilterByPermissions[T1 any, T2 Principal](
	ctx context.Context,
	service PermissionService[T2],
	filter PermissionFilter[T1, T2],
) (T1, error) {
	untypedResult, err := service.Filter(ctx, func(auth Authentication[T2]) (any, error) { return filter(auth) })
	if err != nil {
		var empty T1
		return empty, err
	}

	return untypedResult.(T1), nil
}

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

func (p permissionService[T]) Filter(ctx context.Context, filter UntypedPermissionFilter[T]) (any, error) {
	auth, ok := GetAuthentication[T](ctx)
	if !ok {
		return nil, errors.New("authentication not found")
	}

	return filter(auth)
}
