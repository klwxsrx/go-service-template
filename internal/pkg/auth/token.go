package auth

import (
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/auth"
)

const (
	PrincipalTypeUser      auth.PrincipalType = "user"
	PrincipalTypeAdminUser auth.PrincipalType = "adminUser"
	PrincipalTypeService   auth.PrincipalType = "service"
)

type (
	UserIDToken struct {
		ID uuid.UUID
	}

	AdminUserIDToken struct {
		ID string
	}

	ServiceNameToken struct {
		Name ServiceName
	}

	ServiceName string
)

func (t UserIDToken) Type() auth.PrincipalType {
	return PrincipalTypeUser
}

func (t AdminUserIDToken) Type() auth.PrincipalType {
	return PrincipalTypeAdminUser
}

func (t ServiceNameToken) Type() auth.PrincipalType {
	return PrincipalTypeService
}
