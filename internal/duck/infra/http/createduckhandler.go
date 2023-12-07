package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/duck/api"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type createDuckHandler struct {
	duckService api.DuckService
}

func NewCreateDuckHandler(duckService api.DuckService) pkghttp.Handler {
	return createDuckHandler{duckService: duckService}
}

func (h createDuckHandler) Method() string {
	return http.MethodPost
}

func (h createDuckHandler) Path() string {
	return "/duck"
}

func (h createDuckHandler) HTTPHandler() pkghttp.HandlerFunc {
	return func(w pkghttp.ResponseWriter, r *http.Request) (err error) {
		data, err := pkghttp.Parse(r, pkghttp.JSONBody[createDuckIn](), err)
		_ = pkghttp.ParseOptional(r, pkghttp.QueryParameter[*string]("utm"), err)
		if err != nil {
			return err
		}

		duckID, err := h.duckService.Create(r.Context(), data.Name)
		if err != nil {
			return err
		}

		w.SetJSONBody(createDuckOut{DuckID: duckID})
		return nil
	}
}

type createDuckIn struct {
	Name string `json:"name"`
}

type createDuckOut struct {
	DuckID uuid.UUID `json:"id"`
}
