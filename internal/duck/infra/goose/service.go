package goose

import (
	"github.com/klwxsrx/go-service-template/internal/duck/app/external"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type service struct {
	httpClient pkghttp.Client
}

func (g service) DoSome() error {
	return nil
}

func NewService(httpClient pkghttp.Client) external.GooseService {
	return service{httpClient: httpClient}
}
