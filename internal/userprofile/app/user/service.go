package user

import (
	"context"
	"errors"
	"time"

	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
)

var ErrUserNotFound = errors.New("user not found")

type (
	Service interface {
		Get(context.Context, domain.UserID) (*Data, error)
	}

	Data struct {
		ID        domain.UserID
		DeletedAt *time.Time
	}
)
