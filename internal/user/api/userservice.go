package api

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
)

var (
	ErrInvalidUserCredentials = service.ErrInvalidUserCredentials
	ErrUserNotFound           = service.ErrUserNotFound
	ErrUserIsAlreadyDeleted   = service.ErrUserIsAlreadyDeleted
	ErrUserAlreadyExists      = service.ErrUserAlreadyExists
)

type UserService interface {
	GetByID(context.Context, domain.UserID) (*service.UserData, error)
	GetByIDs(context.Context, []domain.UserID) ([]service.UserData, error)
	Register(context.Context, service.UserCredentials) (domain.UserID, error)
	Delete(context.Context, domain.UserID) error
}
