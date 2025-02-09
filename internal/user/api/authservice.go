package api

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
)

type AuthenticationService interface {
	Authenticate(ctx context.Context, login, password string) (service.SessionTokenData, error)
	VerifyAuthentication(context.Context, service.SessionToken) (service.AuthenticationData, error)
}
