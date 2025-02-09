package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type DeleteUserByIDHandler struct {
	userService service.User
}

func NewDeleteUserByIDHandler(userService service.User) DeleteUserByIDHandler {
	return DeleteUserByIDHandler{
		userService: userService,
	}
}

func (h DeleteUserByIDHandler) Method() string {
	return http.MethodDelete
}

func (h DeleteUserByIDHandler) Path() string {
	return "/users/{userID}"
}

func (h DeleteUserByIDHandler) Handle(_ pkghttp.ResponseWriter, r *http.Request) (err error) {
	userID, err := pkghttp.ParseRequest(r, pkghttp.PathParameter[uuid.UUID]("userID"), err)
	if err != nil {
		return err
	}

	return h.userService.Delete(r.Context(), domain.UserID{UUID: userID})
}
