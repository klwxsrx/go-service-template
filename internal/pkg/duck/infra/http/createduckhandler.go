package http

import (
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"net/http"
)

type createDuckHandler struct {
	duckService service.DuckService
}

func (h *createDuckHandler) Method() string {
	return http.MethodPost
}

func (h *createDuckHandler) Path() string {
	return "/duck/"
}

func (h *createDuckHandler) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h.duckService.Create(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func NewCreateDuckHandler(duckService service.DuckService) pkghttp.Handler {
	return &createDuckHandler{duckService: duckService}
}
