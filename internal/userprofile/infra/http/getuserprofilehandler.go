package http

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/userprofile/app/service"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type GetUserProfileHandler struct {
	userProfileService service.UserProfile
	dtoConverter       DTOConverter
}

func NewGetUserProfileHandler(userProfileService service.UserProfile, dtoConverter DTOConverter) GetUserProfileHandler {
	return GetUserProfileHandler{
		userProfileService: userProfileService,
		dtoConverter:       dtoConverter,
	}
}

func (h GetUserProfileHandler) Method() string {
	return http.MethodGet
}

func (h GetUserProfileHandler) Path() string {
	return "/profile/{userID}"
}

func (h GetUserProfileHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) (err error) {
	userID, err := pkghttp.ParseRequest(r, pkghttp.PathParameter[uuid.UUID]("userID"), err)
	if err != nil {
		return err
	}

	userProfile, err := h.userProfileService.Get(r.Context(), domain.UserID{UUID: userID})
	if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrUserProfileNotFound) {
		w.SetStatusCode(http.StatusNotFound)
	}
	if err != nil {
		return err
	}

	w.SetJSONBody(h.dtoConverter.ToHTTPUserProfileOut(userProfile))
	return nil
}

type UserProfileOut struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}
