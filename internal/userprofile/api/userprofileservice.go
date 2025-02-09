package api

import (
	"context"

	"github.com/klwxsrx/go-service-template/internal/userprofile/app/service"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/user"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
)

var (
	ErrUserNotFound           = service.ErrUserNotFound
	ErrUserProfileNotFound    = service.ErrUserProfileNotFound
	ErrInvalidUserProfileData = service.ErrInvalidUserProfileData
)

type UserProfileService interface {
	Get(context.Context, domain.UserID) (*service.UserProfileData, error)
	Update(context.Context, *service.UserProfileData) error
	HandleUserDeleted(context.Context, user.EventUserDeleted) error
}
