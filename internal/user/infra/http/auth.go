package http

import (
	"net/http"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
)

const (
	userSessionCookieName = "ust"
)

func NewUserSessionCookie(token service.SessionTokenData) *http.Cookie {
	return &http.Cookie{
		Name:     userSessionCookieName,
		Value:    string(token.Token),
		Path:     "/",
		Expires:  token.ValidTill,
		HttpOnly: true,
	}
}
