package cmd

import (
	"fmt"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/http"
)

type HTTPClientFactory struct {
	impl http.ClientFactory
}

func (f *HTTPClientFactory) InitRawClient(extraOpts ...http.ClientOption) http.Client {
	return f.impl.InitRawClient(extraOpts...)
}

func (f *HTTPClientFactory) MustInitClient(dest http.Destination, extraOpts ...http.ClientOption) http.Client {
	hostEnv := fmt.Sprintf("%s_SERVICE_URL", strcase.ToScreamingSnake(string(dest)))
	host := env.Must(env.ParseString(hostEnv))

	return f.impl.InitClient(dest, host, extraOpts...)
}
