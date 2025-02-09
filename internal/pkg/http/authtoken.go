package http

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

const (
	HeaderAuthUserID      = "X-Auth-User-ID"
	HeaderAuthAdminUserID = "X-Auth-AdminUser-ID"
	HeaderAuthServiceName = "X-Auth-Service-Name"
)

func UserIDTokenProvider(r *http.Request) (pkgauth.Token, bool) {
	userID, err := pkghttp.ParseRequest(r, pkghttp.Header[uuid.UUID](HeaderAuthUserID), nil)
	if err != nil || userID == uuid.Nil {
		return nil, false
	}

	return auth.UserIDToken{
		ID: userID,
	}, true
}

func AdminUserIDTokenProvider(r *http.Request) (pkgauth.Token, bool) {
	userID, err := pkghttp.ParseRequest(r, pkghttp.Header[string](HeaderAuthAdminUserID), nil)
	if err != nil || userID == "" {
		return nil, false
	}

	return auth.AdminUserIDToken{
		ID: userID,
	}, true
}

func ServiceNameTokenProvider(r *http.Request) (pkgauth.Token, bool) {
	serviceName, err := pkghttp.ParseRequest(r, pkghttp.Header[string](HeaderAuthServiceName), nil)
	if err != nil || serviceName == "" {
		return nil, false
	}

	return auth.ServiceNameToken{
		Name: auth.ServiceName(serviceName),
	}, true
}

func WithServiceNameAuth(serviceName auth.ServiceName) pkghttp.ClientOption {
	return pkghttp.WithRequestHeader(HeaderAuthServiceName, string(serviceName))
}
