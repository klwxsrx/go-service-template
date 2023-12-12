package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/mux"
)

const (
	DefaultServerAddress = ":8080"

	defaultReadTimeout       = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
)

type (
	ServerOption     func(*mux.Router)
	ServerMiddleware func(http.Handler) http.Handler
)

type HandlerRegistry interface {
	Register(handler Handler, opts ...ServerOption)
}

type Server interface {
	Listener(context.Context) error
	HandlerRegistry
}

type server struct {
	srv    *http.Server
	router *mux.Router
}

func NewServer(
	address string,
	opts ...ServerOption,
) Server {
	router := withHandlerMetadata(mux.NewRouter())
	for _, opt := range opts {
		opt(router)
	}

	srv := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadTimeout:       defaultReadTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	return server{
		srv:    srv,
		router: router,
	}
}

func (s server) Listener(ctx context.Context) error {
	shutdown := func() error {
		err := s.srv.Shutdown(context.Background())
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	}

	serverDoneChan := make(chan error, 1)
	go func() {
		err := s.srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverDoneChan <- err
	}()

	var err error
	select {
	case err = <-serverDoneChan:
	case <-ctx.Done():
		err = shutdown()
	}
	if err != nil {
		return fmt.Errorf("http listener %s: %w", s.srv.Addr, err)
	}

	return nil
}

func (s server) Register(handler Handler, opts ...ServerOption) {
	router := s.router
	if len(opts) > 0 {
		router = s.router.NewRoute().Subrouter()
		for _, opt := range opts {
			opt(router)
		}
	}

	httpHandler := httpHandlerWrapper(handler.HTTPHandler())
	router.
		Name(getRouteName(handler.Method(), handler.Path())).
		Methods(handler.Method()).
		Path(handler.Path()).
		Handler(httpHandler)
}

func getRouteName(method, path string) string {
	path = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
			return r
		}

		if r == '{' || r == '}' {
			return -1
		}

		return '_'
	}, strings.Trim(path, "/"))
	return fmt.Sprintf("%s_%s", strings.ToUpper(method), path)
}
