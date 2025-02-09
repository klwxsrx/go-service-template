package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type GetUserByIDHandler struct {
	userService  service.User
	dtoConverter DTOConverter
}

func NewGetUserByIDHandler(userService service.User, dtoConverter DTOConverter) GetUserByIDHandler {
	return GetUserByIDHandler{
		userService:  userService,
		dtoConverter: dtoConverter,
	}
}

func (h GetUserByIDHandler) Method() string {
	return http.MethodGet
}

func (h GetUserByIDHandler) Path() string {
	return "/users/{userID}"
}

func (h GetUserByIDHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) (err error) {
	userID, err := pkghttp.ParseRequest(r, pkghttp.PathParameter[uuid.UUID]("userID"), err)
	if err != nil {
		return err
	}

	result, err := h.userService.GetByID(r.Context(), domain.UserID{UUID: userID})
	if errors.Is(err, service.ErrUserNotFound) {
		w.SetStatusCode(http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	w.SetJSONBody(h.dtoConverter.ToHTTPUserOut(result))
	return nil
}

type UserOut struct {
	ID        uuid.UUID  `json:"id"`
	Login     string     `json:"login"`
	DeletedAt *time.Time `json:"deletedAt"`
}
