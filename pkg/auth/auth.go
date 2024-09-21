package auth

import (
	"context"
	"errors"
)

var ErrUnauthenticated = errors.New("not authenticated")

type (
	Provider[T Principal] interface {
		Authenticate(context.Context, Token) (Authentication[T], error)
	}

	Token interface {
		Type() PrincipalType
	}

	Authentication[T Principal] interface {
		IsAuthenticated() bool
		Principal() *T
	}

	Principal interface {
		Type() PrincipalType
		ID() *string
	}

	Auth[T Principal] struct {
		AuthPrincipal *T
	}

	PrincipalType string
)

func (a Auth[T]) IsAuthenticated() bool {
	return a.AuthPrincipal != nil
}

func (a Auth[T]) Principal() *T {
	return a.AuthPrincipal
}
