package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/auth"
)

type (
	Principal struct {
		UserID      *uuid.UUID
		AdminUserID *string
		ServiceName *ServiceName
	}

	provider struct{}
)

func NewProvider() auth.Provider[Principal] {
	return provider{}
}

func (p provider) Authenticate(_ context.Context, token auth.Token) (auth.Authentication[Principal], error) {
	var principal *Principal
	switch t := token.(type) {
	case UserIDToken:
		principal = &Principal{UserID: &t.ID}
	case AdminUserIDToken:
		principal = &Principal{AdminUserID: &t.ID}
	case ServiceNameToken:
		principal = &Principal{ServiceName: &t.Name}
	default:
		return nil, fmt.Errorf("unknown token with type %s", token.Type())
	}

	return auth.Auth[Principal]{AuthPrincipal: principal}, nil
}

func (p Principal) Type() auth.PrincipalType {
	switch {
	case p.UserID != nil:
		return PrincipalTypeUser
	case p.AdminUserID != nil:
		return PrincipalTypeAdminUser
	case p.ServiceName != nil:
		return PrincipalTypeService
	default:
		return "unknown"
	}
}

func (p Principal) ID() *string {
	switch {
	case p.UserID != nil:
		v := p.UserID.String()
		return &v
	case p.AdminUserID != nil:
		return p.AdminUserID
	case p.ServiceName != nil:
		return (*string)(p.ServiceName)
	default:
		return nil
	}
}
