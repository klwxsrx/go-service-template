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
	return func(w pkghttp.ResponseWriter, r *http.Request) {
		data, err := pkghttp.Parse(pkghttp.JSONBody[CreateDuckRequest](), r, nil)
		if err != nil {
			w.SetStatusCode(http.StatusBadRequest)
			return
		}

		err = h.duckService.Create(r.Context(), data.Name)
		if err != nil {
			w.SetStatusCode(http.StatusInternalServerError)
			return
		}

		w.SetStatusCode(http.StatusCreated)
	}
}

func NewCreateDuckHandler(duckService service.DuckService) pkghttp.Handler {
	return &createDuckHandler{duckService: duckService}
}

type CreateDuckRequest struct {
	Name string `json:"name"`
}
