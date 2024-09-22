package http

import (
	"context"
	"net/http"

	"github.com/klwxsrx/go-service-template/pkg/auth"
)

type AuthTokenProvider func(*http.Request) (auth.Token, bool)

func WithAuth[T auth.Principal](provider auth.Provider[T], tokenProviders ...AuthTokenProvider) ServerOption {
	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var ok bool
			var token auth.Token
			for _, tokenProvider := range tokenProviders {
				token, ok = tokenProvider(r)
				if ok {
					break
				}
			}
			if !ok {
				r = setHandlerAuthentication(r, auth.Auth[T]{})
				handler.ServeHTTP(w, r)
				return
			}

			authData, err := provider.Authenticate(r.Context(), token)
			if err != nil {
				writeHandlerResult(r.Context(), w, http.StatusInternalServerError, err)
				return
			}

			r = setHandlerAuthentication(r, authData)
			handler.ServeHTTP(w, r)
		})
	})
}

func WithAuthenticationRequirement() ServerOption {
	return WithMW(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAuthenticated, err := auth.IsAuthenticated(r.Context())
			if err != nil {
				writeHandlerResult(r.Context(), w, http.StatusInternalServerError, err)
				return
			}

			if !isAuthenticated {
				writeHandlerResult(r.Context(), w, http.StatusUnauthorized, auth.ErrUnauthenticated)
				return
			}

			handler.ServeHTTP(w, r)
		})
	})
}

func setHandlerAuthentication[T auth.Principal](r *http.Request, a auth.Authentication[T]) *http.Request {
	var principal *auth.Principal
	if a.Principal() != nil {
		p := auth.Principal(*a.Principal())
		principal = &p
	}

	meta := getHandlerMetadata(r.Context())
	meta.Auth = auth.Auth[auth.Principal]{AuthPrincipal: principal}

	return r.WithContext(auth.WithAuthentication(r.Context(), a))
}

func writeHandlerResult(ctx context.Context, w http.ResponseWriter, httpCode int, err error) {
	meta := getHandlerMetadata(ctx)
	meta.Code = httpCode
	meta.Error = err

	w.WriteHeader(httpCode)
}
