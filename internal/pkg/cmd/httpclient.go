package cmd

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/strings"
)

type HTTPClientFactory struct {
	impl http.ClientFactory
}

func NewHTTPClientFactory(
	opts ...http.ClientOption,
) HTTPClientFactory {
	return HTTPClientFactory{
		impl: http.NewClientFactory(opts...),
	}
}

func (f HTTPClientFactory) InitRawClient(extraOpts ...http.ClientOption) http.Client {
	return f.impl.InitRawClient(extraOpts...)
}

func (f HTTPClientFactory) MustInitClient(dest http.Destination, extraOpts ...http.ClientOption) http.Client {
	hostEnv := fmt.Sprintf("%s_SERVICE_URL", strings.ToScreamingSnakeCase(string(dest)))
	host := env.Must(env.Parse[string](hostEnv))

	return f.impl.InitClient(dest, host, extraOpts...)
}
