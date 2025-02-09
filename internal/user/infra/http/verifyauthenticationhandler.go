package http

import (
	"net/http"

	internalhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

// VerifyAuthenticationHandler is used by API Gateway to authenticate users
// The handler gets the session token cookie and auths users using the token
// Returns 401 if user is not authenticated or the token is invalid or expired
// If authentication is successful the handler will return the internal UserID header
// with an optional renewed session cookie
type VerifyAuthenticationHandler struct {
	authService service.Authentication
}

func NewVerifyAuthenticationHandler(authService service.Authentication) VerifyAuthenticationHandler {
	return VerifyAuthenticationHandler{authService: authService}
}

func (h VerifyAuthenticationHandler) Method() string {
	return http.MethodPost
}

func (h VerifyAuthenticationHandler) Path() string {
	return "/auth/verification"
}

func (h VerifyAuthenticationHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) (err error) {
	sessionToken, err := pkghttp.ParseRequest(r, pkghttp.CookieValue[string](userSessionCookieName), err)
	if err != nil {
		w.SetStatusCode(http.StatusUnauthorized)
		return err
	}

	authData, err := h.authService.VerifyAuthentication(r.Context(), service.SessionToken(sessionToken))
	if err != nil {
		return err
	}

	if authData.RenewedToken != nil {
		w.SetCookie(NewUserSessionCookie(*authData.RenewedToken))
	}

	w.SetHeader(internalhttp.HeaderAuthUserID, authData.UserID.String())
	return nil
}
