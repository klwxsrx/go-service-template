package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/user/app/encoding"
	"github.com/klwxsrx/go-service-template/internal/user/app/permission"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

var (
	ErrInvalidUserCredentials = errors.New("invalid user credentials")
	ErrUserNotFound           = errors.New("user not found")
	ErrUserIsAlreadyDeleted   = errors.New("user is already deleted")
	ErrUserAlreadyExists      = errors.New("user with specified login already exists")
)

const updateUsersLockName = "update_users"

type (
	User interface {
		GetByID(context.Context, domain.UserID) (*UserData, error)
		GetByIDs(context.Context, []domain.UserID) ([]UserData, error)
		Register(context.Context, UserCredentials) (domain.UserID, error)
		Delete(context.Context, domain.UserID) error
	}

	UserCredentials struct {
		Login    string
		Password string
	}

	UserData struct {
		ID        domain.UserID
		Login     string
		DeletedAt *time.Time
	}

	userService struct {
		userRepo        domain.UserRepository
		dtoConverter    DTOConverter
		passwordEncoder encoding.PasswordEncoder
		permissions     auth.PermissionService
		transaction     persistence.Transaction
	}
)

func NewUser(
	userRepo domain.UserRepository,
	passwordEncoder encoding.PasswordEncoder,
	permissions auth.PermissionService,
	transaction persistence.Transaction,
	dtoConverter DTOConverter,
) User {
	return &userService{
		userRepo:        userRepo,
		dtoConverter:    dtoConverter,
		passwordEncoder: passwordEncoder,
		permissions:     permissions,
		transaction:     transaction,
	}
}

func (s *userService) GetByID(ctx context.Context, userID domain.UserID) (*UserData, error) {
	if err := s.permissions.Check(ctx, permission.CanReadUser(userID)); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindOne(ctx, domain.FindUserSpecification{IDs: []domain.UserID{userID}})
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.dtoConverter.ToDTOUserData(user), nil
}

func (s *userService) GetByIDs(ctx context.Context, userIDs []domain.UserID) ([]UserData, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var err error
	if userIDs, err = pkgauth.FilterByPermissions(ctx, s.permissions, permission.CanReadUserFilter(userIDs)); err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		return nil, nil
	}

	users, err := s.userRepo.Find(ctx, domain.FindUserSpecification{IDs: userIDs})
	if err != nil {
		return nil, fmt.Errorf("find users by ids: %w", err)
	}

	return s.dtoConverter.ToDTOUsersData(users), nil
}

func (s *userService) Register(ctx context.Context, credentials UserCredentials) (domain.UserID, error) {
	login := strings.TrimSpace(credentials.Login)
	password := strings.TrimSpace(credentials.Password)
	if login == "" || password == "" {
		return domain.UserID{}, ErrInvalidUserCredentials
	}

	passwordHash, err := s.passwordEncoder.HashPassword(password)
	if err != nil {
		return domain.UserID{}, fmt.Errorf("hash password: %w", err)
	}

	login = strings.ToLower(login)
	registerUserImpl := func(ctx context.Context) (domain.UserID, error) {
		user, err := s.userRepo.FindOne(ctx, domain.FindUserSpecification{Logins: []string{login}})
		if errors.Is(err, domain.ErrUserNotFound) {
			userID := s.userRepo.NextID()
			err = s.userRepo.Store(ctx, &domain.User{
				ID:           userID,
				Login:        login,
				PasswordHash: passwordHash,
				DeletedAt:    nil,
			})
			if err != nil {
				return domain.UserID{}, fmt.Errorf("store user: %w", err)
			}

			return userID, nil
		}
		if err != nil {
			return domain.UserID{}, fmt.Errorf("find user by login: %w", err)
		}

		if user.DeletedAt != nil {
			return domain.UserID{}, ErrUserIsAlreadyDeleted
		}
		if user.PasswordHash != passwordHash {
			return domain.UserID{}, ErrUserAlreadyExists
		}

		return user.ID, nil
	}

	return persistence.WithinTransactionWithResult(ctx, s.transaction, registerUserImpl, updateUsersLockName)
}

func (s *userService) Delete(ctx context.Context, userID domain.UserID) error {
	if err := s.permissions.Check(ctx, permission.CanDeleteUser(userID)); err != nil {
		return err
	}

	return s.transaction.WithinContext(ctx, func(ctx context.Context) error {
		user, err := s.userRepo.FindOne(s.transaction.WithLock(ctx), domain.FindUserSpecification{IDs: []domain.UserID{userID}})
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("find user by id: %w", err)
		}

		if user.DeletedAt != nil {
			return nil
		}

		now := time.Now()
		user.SetDeletedAt(now)

		err = s.userRepo.Store(ctx, user)
		if err != nil {
			return fmt.Errorf("store user: %w", err)
		}

		return nil
	})
}
