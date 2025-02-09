package http

import (
	"net/http"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type GetCurrentUserHandler struct {
	userService  service.User
	dtoConverter DTOConverter
}

func NewGetCurrentUserHandler(userService service.User, dtoConverter DTOConverter) GetCurrentUserHandler {
	return GetCurrentUserHandler{
		userService:  userService,
		dtoConverter: dtoConverter,
	}
}

func (h GetCurrentUserHandler) Method() string {
	return http.MethodGet
}

func (h GetCurrentUserHandler) Path() string {
	return "/current-user"
}

func (h GetCurrentUserHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) error {
	authentication, ok := pkgauth.GetAuthentication[auth.Principal](r.Context())
	if !ok || authentication.Principal() == nil || authentication.Principal().UserID == nil {
		return pkgauth.ErrUnauthenticated
	}

	result, err := h.userService.GetByID(r.Context(), domain.UserID{UUID: *authentication.Principal().UserID})
	if err != nil {
		return err
	}

	w.SetJSONBody(h.dtoConverter.ToHTTPUserOut(result))
	return nil
}
