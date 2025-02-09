package session

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/klwxsrx/go-service-template/internal/user/app/session"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
)

// tokenGenerator is a fake example implementation
// Use JWT token implementation in a real application
type tokenGenerator struct{}

func NewTokenGenerator() session.TokenGenerator {
	return tokenGenerator{}
}

func (t tokenGenerator) Generate(userID domain.UserID, ttl time.Duration) (session.TokenData, error) {
	createdAt := time.Now()
	data := tokenData{
		UserID:    userID,
		CreatedAt: createdAt,
		ValidTill: createdAt.Add(ttl),
	}

	encodedData, err := json.Marshal(data)
	if err != nil {
		return session.TokenData{}, fmt.Errorf("encode token data to json: %w", err)
	}

	token := session.EncodedToken(base64.StdEncoding.EncodeToString(encodedData))
	return session.TokenData{
		EncodedToken: token,
		UserID:       data.UserID,
		CreatedAt:    data.CreatedAt,
		ValidTill:    data.ValidTill,
	}, nil
}

func (t tokenGenerator) Decode(token session.EncodedToken) (session.TokenData, error) {
	tokenJSON, err := base64.StdEncoding.DecodeString(string(token))
	if err != nil {
		return session.TokenData{}, fmt.Errorf("%w: decode token base64: %w", session.ErrInvalidToken, err)
	}

	var data tokenData
	err = json.Unmarshal(tokenJSON, &data)
	if err != nil {
		return session.TokenData{}, fmt.Errorf("%w: decode token data from json: %w", session.ErrInvalidToken, err)
	}
	if data.ValidTill.Before(time.Now()) {
		return session.TokenData{}, fmt.Errorf("%w: token is expired", session.ErrInvalidToken)
	}

	return session.TokenData{
		EncodedToken: token,
		UserID:       data.UserID,
		CreatedAt:    data.CreatedAt,
		ValidTill:    data.ValidTill,
	}, nil
}

type tokenData struct {
	UserID    domain.UserID `json:"userID"`
	CreatedAt time.Time     `json:"createdAt"`
	ValidTill time.Time     `json:"validTill"`
}
