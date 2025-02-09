package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/permission"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/user"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	"github.com/klwxsrx/go-service-template/pkg/idk"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrUserProfileNotFound    = errors.New("user profile not found")
	ErrInvalidUserProfileData = errors.New("invalid user profile data")
)

type (
	UserProfile interface {
		Get(context.Context, domain.UserID) (*UserProfileData, error)
		Update(context.Context, *UserProfileData) error
		HandleUserDeleted(context.Context, user.EventUserDeleted) error
	}

	UserProfileData struct {
		ID        domain.UserID
		FirstName string
		LastName  string
	}

	userProfileService struct {
		userService user.Service
		profileRepo domain.UserProfileRepository
		permissions auth.PermissionService
		idkService  idk.Service
		transaction persistence.Transaction
		converter   DTOConverter
	}
)

func NewUserProfile(
	userService user.Service,
	profileRepo domain.UserProfileRepository,
	permissions auth.PermissionService,
	idkService idk.Service,
	transaction persistence.Transaction,
	converter DTOConverter,
) UserProfile {
	return &userProfileService{
		userService: userService,
		profileRepo: profileRepo,
		idkService:  idkService,
		permissions: permissions,
		transaction: transaction,
		converter:   converter,
	}
}

func (s *userProfileService) Get(ctx context.Context, id domain.UserID) (*UserProfileData, error) {
	if err := s.permissions.Check(ctx, permission.CanGetUserProfile(id)); err != nil {
		return nil, err
	}

	profile, err := s.profileRepo.FindByID(ctx, id)
	if errors.Is(err, domain.ErrUserProfileNotFound) {
		return nil, ErrUserProfileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find userprofile by id: %w", err)
	}

	userData, err := s.userService.Get(ctx, id)
	if errors.Is(err, user.ErrUserNotFound) || err == nil && userData.DeletedAt != nil {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user from userservice: %w", err)
	}

	return s.converter.ToDTOUserProfileData(profile), nil
}

func (s *userProfileService) Update(ctx context.Context, data *UserProfileData) error {
	if err := s.permissions.Check(ctx, permission.CanUpdateUserProfile(data.ID)); err != nil {
		return err
	}

	firstName := strings.TrimSpace(data.FirstName)
	lastName := strings.TrimSpace(data.LastName)
	if firstName == "" || lastName == "" {
		return ErrInvalidUserProfileData
	}

	userData, err := s.userService.Get(ctx, data.ID)
	if errors.Is(err, user.ErrUserNotFound) || err == nil && userData.DeletedAt != nil {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("find user from userservice: %w", err)
	}

	err = s.profileRepo.Store(ctx, &domain.UserProfile{
		ID:        data.ID,
		FirstName: data.FirstName,
		LastName:  data.LastName,
	})
	if err != nil {
		return fmt.Errorf("store userprofile: %w", err)
	}

	return nil
}

func (s *userProfileService) HandleUserDeleted(ctx context.Context, event user.EventUserDeleted) error {
	userData, err := s.userService.Get(ctx, event.UserID)
	if errors.Is(err, user.ErrUserNotFound) || err == nil && userData.DeletedAt == nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user from userservice: %w", err)
	}

	return s.transaction.WithinContext(ctx, func(ctx context.Context) error {
		if err = s.idkService.Insert(ctx, event.EventID, "handle_user_deleted"); err != nil {
			return err
		}

		err = s.profileRepo.DeleteByID(ctx, event.UserID)
		if errors.Is(err, domain.ErrUserProfileNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("delete userprofile by id: %w", err)
		}

		return nil
	})
}
