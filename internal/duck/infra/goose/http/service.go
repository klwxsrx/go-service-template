package http

import (
	"github.com/klwxsrx/go-service-template/internal/duck/app/goose"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

type service struct{}

func NewService(_ pkghttp.Client) goose.Service {
	return service{}
}

func (g service) DoSome() error {
	return nil
}
