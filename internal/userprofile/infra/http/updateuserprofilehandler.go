package http

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/userprofile/app/service"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type UpdateUserProfileHandler struct {
	userProfileService service.UserProfile
}

func NewUpdateUserProfileHandler(userProfileService service.UserProfile) UpdateUserProfileHandler {
	return UpdateUserProfileHandler{
		userProfileService: userProfileService,
	}
}

func (h UpdateUserProfileHandler) Method() string {
	return http.MethodPut
}

func (h UpdateUserProfileHandler) Path() string {
	return "/profile/{userID}"
}

func (h UpdateUserProfileHandler) Handle(w pkghttp.ResponseWriter, r *http.Request) (err error) {
	userID, err := pkghttp.ParseRequest(r, pkghttp.PathParameter[uuid.UUID]("userID"), err)
	in, err := pkghttp.ParseRequest(r, pkghttp.JSONBody[updateUserProfileIn](), err)
	if err != nil {
		return err
	}

	err = h.userProfileService.Update(r.Context(), &service.UserProfileData{
		ID:        domain.UserID{UUID: userID},
		FirstName: in.FirstName,
		LastName:  in.LastName,
	})
	switch {
	case errors.Is(err, service.ErrInvalidUserProfileData):
		w.SetStatusCode(http.StatusBadRequest)
	case errors.Is(err, service.ErrUserNotFound):
		w.SetStatusCode(http.StatusNotFound)
	}

	return err
}

type updateUserProfileIn struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}
