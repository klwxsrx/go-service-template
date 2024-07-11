package auth

import (
	"context"
	"fmt"
)

type (
	Service[T Principal] interface {
		Authenticate(context.Context, Token) (Authentication[T], error)
	}

	Provider[T Principal] interface {
		IsSupportedFor(TokenType) bool
		Authenticate(context.Context, Token) (Authentication[T], error)
	}

	Token interface {
		Type() TokenType
	}

	Principal interface {
		Type() PrincipalType
		ID() *string
	}

	Authentication[T Principal] interface {
		IsAuthenticated() bool
		Principal() *T
	}

	Auth[T Principal] struct {
		AuthPrincipal *T
	}

	TokenType     string
	PrincipalType string

	service[T Principal] struct {
		providers []Provider[T]
	}
)

func NewService[T Principal](providers ...Provider[T]) Service[T] {
	return service[T]{
		providers: providers,
	}
}

func (s service[T]) Authenticate(ctx context.Context, token Token) (Authentication[T], error) {
	for _, provider := range s.providers {
		if !provider.IsSupportedFor(token.Type()) {
			continue
		}

		auth, err := provider.Authenticate(ctx, token)
		if err != nil {
			return auth, fmt.Errorf("auth token with type %s: %w", token.Type(), err)
		}

		return auth, nil
	}

	return Auth[T]{}, nil
}

func (a Auth[T]) IsAuthenticated() bool {
	return a.AuthPrincipal != nil
}

func (a Auth[T]) Principal() *T {
	return a.AuthPrincipal
}
