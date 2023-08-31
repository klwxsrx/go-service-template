package http

import (
	"net/http"

	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type createDuckHandler struct {
	duckService service.DuckService
}

func (h *createDuckHandler) Method() string {
	return http.MethodPost
}

func (h *createDuckHandler) Path() string {
	return "/duck"
}

func (h *createDuckHandler) HTTPHandler() pkghttp.HandlerFunc {
	return func(w pkghttp.ResponseWriter, r *http.Request) (err error) {
		data, err := pkghttp.Parse(pkghttp.JSONBody[createDuckRequest](), r, err)
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

func NewCreateDuckHandler(duckService service.DuckService) pkghttp.Handler {
	return &createDuckHandler{duckService: duckService}
}

type createDuckRequest struct {
	Name string `json:"name"`
}
