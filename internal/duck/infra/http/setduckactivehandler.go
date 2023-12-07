package http

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/duck/api"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type setDuckActiveHandler struct {
	duckService api.DuckService
}

func NewSetDuckActiveHandler(duckService api.DuckService) pkghttp.Handler {
	return setDuckActiveHandler{duckService: duckService}
}

func (h setDuckActiveHandler) Method() string {
	return http.MethodPost
}

func (h setDuckActiveHandler) Path() string {
	return "/duck/{duckID}/setActive/{isActive}"
}

func (h setDuckActiveHandler) HTTPHandler() pkghttp.HandlerFunc {
	return func(w pkghttp.ResponseWriter, r *http.Request) (err error) {
		duckID, err := pkghttp.Parse(r, pkghttp.PathParameter[uuid.UUID]("duckID"), nil)
		isActive, err := pkghttp.Parse(r, pkghttp.PathParameter[bool]("isActive"), err)
		if err != nil {
			return err
		}

		err = h.duckService.SetActive(r.Context(), duckID, isActive)
		if errors.Is(err, api.ErrDuckNotFound) {
			w.SetStatusCode(http.StatusNotFound)
			return nil
		}
		if err != nil {
			return err
		}

		return nil
	}
}
