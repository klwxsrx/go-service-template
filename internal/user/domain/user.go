package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/event"
)

var ErrUserNotFound = errors.New("user not found")

type (
	User struct {
		ID           UserID
		Login        string
		PasswordHash string
		DeletedAt    *time.Time

		Changes []event.Event
	}

	UserRepository interface {
		NextID() UserID
		Store(context.Context, *User) error
		Find(context.Context, FindUserSpecification) ([]User, error)
		FindOne(context.Context, FindUserSpecification) (*User, error)
	}

	FindUserSpecification struct {
		IDs    []UserID
		Logins []string
	}

	UserID struct{ uuid.UUID }
)

func (u *User) SetDeletedAt(t time.Time) {
	u.DeletedAt = &t
	u.Changes = append(u.Changes, EventUserDeleted{
		EventID: uuid.New(),
		UserID:  u.ID,
	})
}
