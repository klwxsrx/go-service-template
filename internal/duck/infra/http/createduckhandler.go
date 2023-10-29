package http

import (
	"net/http"

	"github.com/klwxsrx/go-service-template/internal/duck/api"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

const utmSourceQueryParamName = "utm"

type createDuckHandler struct {
	duckService api.DuckService
}

func (h *createDuckHandler) Method() string {
	return http.MethodPost
}

func (h *createDuckHandler) Path() string {
	return "/duck"
}

func (h *createDuckHandler) HTTPHandler() pkghttp.HandlerFunc {
	return func(w pkghttp.ResponseWriter, r *http.Request) (err error) {
		data, err := pkghttp.Parse(r, pkghttp.JSONBody[createDuckRequest](), err)
		_ = pkghttp.ParseOptional(r, pkghttp.QueryParameter[*string](utmSourceQueryParamName), err)
		if err != nil {
			return err
		}

		err = h.duckService.Create(r.Context(), data.Name)
		if err != nil {
			return err
		}

		w.SetStatusCode(http.StatusCreated)
		return nil
	}
}

func NewCreateDuckHandler(duckService api.DuckService) pkghttp.Handler {
	return &createDuckHandler{duckService: duckService}
}

type createDuckRequest struct {
	Name string `json:"name"`
}
