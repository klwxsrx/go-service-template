package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type RegisterUserHandler struct {
	userService  service.User
	dtoConverter DTOConverter
}

func NewRegisterUserHandler(userService service.User, dtoConverter DTOConverter) RegisterUserHandler {
	return RegisterUserHandler{userService: userService, dtoConverter: dtoConverter}
}

func (h RegisterUserHandler) Method() string {
	return http.MethodPost
}

func (h RegisterUserHandler) Path() string {
	return "/users"
}

func (h RegisterUserHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) (err error) {
	in, err := pkghttp.ParseRequest(r, pkghttp.JSONBody[RegisterUserIn](), err)
	if err != nil {
		return err
	}

	in.Login = strings.TrimSpace(in.Login)
	if in.Login == "" {
		w.SetStatusCode(http.StatusBadRequest)
		return fmt.Errorf("login must be not empty")
	}

	in.Password = strings.TrimSpace(in.Password)
	if in.Password == "" {
		w.SetStatusCode(http.StatusBadRequest)
		return fmt.Errorf("password must be not empty")
	}

	userID, err := h.userService.Register(r.Context(), h.dtoConverter.ToDTOUserCredentials(in))
	if errors.Is(err, service.ErrInvalidUserCredentials) {
		w.SetStatusCode(http.StatusBadRequest)
		return err
	}
	if errors.Is(err, service.ErrUserIsAlreadyDeleted) {
		w.SetStatusCode(http.StatusForbidden)
		return err
	}
	if errors.Is(err, service.ErrUserAlreadyExists) {
		w.SetStatusCode(http.StatusConflict)
	}
	if err != nil {
		return err
	}

	w.SetJSONBody(registerUserOut{ID: userID.UUID})
	w.SetStatusCode(http.StatusCreated)
	return nil
}

type (
	RegisterUserIn struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	registerUserOut struct {
		ID uuid.UUID `json:"id"`
	}
)
