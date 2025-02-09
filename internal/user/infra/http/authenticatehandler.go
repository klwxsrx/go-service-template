package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type AuthenticateHandler struct {
	authService service.Authentication
}

func NewAuthenticateHandler(authService service.Authentication) AuthenticateHandler {
	return AuthenticateHandler{authService: authService}
}

func (h AuthenticateHandler) Method() string {
	return http.MethodPost
}

func (h AuthenticateHandler) Path() string {
	return "/auth"
}

func (h AuthenticateHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) (err error) {
	in, err := pkghttp.ParseRequest(r, pkghttp.JSONBody[authenticateIn](), err)
	if err != nil {
		w.SetStatusCode(http.StatusUnauthorized)
		return err
	}

	in.Login = strings.TrimSpace(in.Login)
	if in.Login == "" {
		w.SetStatusCode(http.StatusUnauthorized)
		return fmt.Errorf("login must be not empty")
	}

	in.Password = strings.TrimSpace(in.Password)
	if in.Password == "" {
		w.SetStatusCode(http.StatusUnauthorized)
		return fmt.Errorf("password must be not empty")
	}

	token, err := h.authService.Authenticate(r.Context(), in.Login, in.Password)
	if err != nil {
		return err
	}

	w.SetCookie(NewUserSessionCookie(token))
	return nil
}

type authenticateIn struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
