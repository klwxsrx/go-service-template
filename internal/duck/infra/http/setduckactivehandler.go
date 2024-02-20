package http

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/duck/api"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type SetDuckActiveHandler struct {
	api api.API
}

func NewSetDuckActiveHandler(api api.API) SetDuckActiveHandler {
	return SetDuckActiveHandler{api: api}
}

func (h SetDuckActiveHandler) Method() string {
	return http.MethodPost
}

func (h SetDuckActiveHandler) Path() string {
	return "/duck/{duckID}/setActive/{isActive}"
}

func (h SetDuckActiveHandler) HTTPHandler() pkghttp.HandlerFunc {
	return func(w pkghttp.ResponseWriter, r *http.Request) (err error) {
		duckID, err := pkghttp.Parse(r, pkghttp.PathParameter[uuid.UUID]("duckID"), nil)
		isActive, err := pkghttp.Parse(r, pkghttp.PathParameter[bool]("isActive"), err)
		if err != nil {
			return err
		}

		err = h.api.SetActive(r.Context(), duckID, isActive)
		if errors.Is(err, api.ErrDuckNotFound) {
			w.SetStatusCode(http.StatusNotFound)
			return nil
		}
		return err
	}
}
