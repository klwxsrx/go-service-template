package session

import (
	"errors"
	"time"

	"github.com/klwxsrx/go-service-template/internal/user/domain"
)

var ErrInvalidToken = errors.New("token is invalid or expired")

type (
	TokenGenerator interface {
		Generate(userID domain.UserID, ttl time.Duration) (TokenData, error)
		Decode(EncodedToken) (TokenData, error)
	}

	TokenData struct {
		EncodedToken EncodedToken
		UserID       domain.UserID
		CreatedAt    time.Time
		ValidTill    time.Time
	}

	EncodedToken string
)
