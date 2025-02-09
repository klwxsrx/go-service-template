package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/klwxsrx/go-service-template/internal/user/app/encoding"
	"github.com/klwxsrx/go-service-template/internal/user/app/session"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	"github.com/klwxsrx/go-service-template/pkg/auth"
)

const (
	userSessionTTL             = 7 * 24 * time.Hour
	userSessionRenewalInterval = 15 * time.Minute
)

type (
	Authentication interface {
		Authenticate(ctx context.Context, login, password string) (SessionTokenData, error)
		VerifyAuthentication(context.Context, SessionToken) (AuthenticationData, error)
	}

	SessionTokenData struct {
		Token     SessionToken
		ValidTill time.Time
	}

	AuthenticationData struct {
		UserID       domain.UserID
		RenewedToken *SessionTokenData
	}

	SessionToken string

	authenticationService struct {
		userRepo        domain.UserRepository
		sessionTokens   session.TokenGenerator
		passwordEncoder encoding.PasswordEncoder
	}
)

func NewAuthentication(
	userRepo domain.UserRepository,
	sessionTokens session.TokenGenerator,
	passwordEncoder encoding.PasswordEncoder,
) Authentication {
	return &authenticationService{
		userRepo:        userRepo,
		sessionTokens:   sessionTokens,
		passwordEncoder: passwordEncoder,
	}
}

func (s authenticationService) Authenticate(ctx context.Context, login, password string) (SessionTokenData, error) {
	login = strings.TrimSpace(login)
	password = strings.TrimSpace(password)
	if login == "" || password == "" {
		return SessionTokenData{}, auth.ErrUnauthenticated
	}

	login = strings.ToLower(login)
	user, err := s.userRepo.FindOne(ctx, domain.FindUserSpecification{Logins: []string{login}})
	if errors.Is(err, domain.ErrUserNotFound) {
		return SessionTokenData{}, auth.ErrUnauthenticated
	}
	if err != nil {
		return SessionTokenData{}, fmt.Errorf("find user by login: %w", err)
	}

	if user.DeletedAt != nil || !s.passwordEncoder.CompareHash(user.PasswordHash, password) {
		return SessionTokenData{}, auth.ErrUnauthenticated
	}

	return s.generateSessionToken(user.ID)
}

func (s authenticationService) VerifyAuthentication(ctx context.Context, token SessionToken) (AuthenticationData, error) {
	tokenData, err := s.sessionTokens.Decode(session.EncodedToken(token))
	if errors.Is(err, session.ErrInvalidToken) {
		return AuthenticationData{}, auth.ErrUnauthenticated
	}
	if err != nil {
		return AuthenticationData{}, fmt.Errorf("decode token: %w", err)
	}

	if tokenData.CreatedAt.After(time.Now().Add(-userSessionRenewalInterval)) {
		return AuthenticationData{UserID: tokenData.UserID}, nil
	}

	user, err := s.userRepo.FindOne(ctx, domain.FindUserSpecification{IDs: []domain.UserID{tokenData.UserID}})
	if errors.Is(err, domain.ErrUserNotFound) || err == nil && user.DeletedAt != nil {
		return AuthenticationData{}, auth.ErrUnauthenticated
	}
	if err != nil {
		return AuthenticationData{}, fmt.Errorf("find user by id: %w", err)
	}

	renewedToken, err := s.generateSessionToken(tokenData.UserID)
	if err != nil {
		return AuthenticationData{}, err
	}

	return AuthenticationData{
		UserID:       tokenData.UserID,
		RenewedToken: &renewedToken,
	}, nil
}

func (s authenticationService) generateSessionToken(userID domain.UserID) (SessionTokenData, error) {
	token, err := s.sessionTokens.Generate(userID, userSessionTTL)
	if err != nil {
		return SessionTokenData{}, fmt.Errorf("generate session token: %w", err)
	}

	return SessionTokenData{
		Token:     SessionToken(token.EncodedToken),
		ValidTill: token.ValidTill,
	}, nil
}
