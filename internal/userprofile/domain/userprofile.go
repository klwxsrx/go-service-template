package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrUserProfileNotFound = errors.New("userprofile not found")

type (
	UserProfile struct {
		ID        UserID
		FirstName string
		LastName  string
	}

	UserProfileRepository interface {
		Store(context.Context, *UserProfile) error
		FindByID(context.Context, UserID) (*UserProfile, error)
		DeleteByID(context.Context, UserID) error
	}

	UserID struct{ uuid.UUID }
)
