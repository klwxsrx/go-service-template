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
	defaultServerAddress     = ":8080"
	defaultReadHeaderTimeout = 3 * time.Second
	defaultReadTimeout       = 5 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

type (
	ServerOption      func(*ServerImpl)
	HandlerOption     func(*mux.Router)
	HandlerMiddleware func(http.Handler) http.Handler
)

type HandlerRegistry interface {
	Register(handler Handler, opts ...HandlerOption)
}

type Server interface {
	Listener(context.Context) error
	HandlerRegistry
}

type ServerImpl struct {
	Impl            *http.Server
	ShutdownTimeout time.Duration
}

func NewServer(opts ...ServerOption) Server {
	srv := &ServerImpl{
		Impl: &http.Server{
			Addr:              defaultServerAddress,
			Handler:           withHandlerMetadata(mux.NewRouter()),
			ReadTimeout:       defaultReadTimeout,
			ReadHeaderTimeout: defaultReadHeaderTimeout,
		},
		ShutdownTimeout: defaultShutdownTimeout,
	}
	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

func (s ServerImpl) Listener(ctx context.Context) error {
	shutdown := func() error {
		if s.ShutdownTimeout > 0 {
			var ctxCancel context.CancelFunc
			ctx, ctxCancel = context.WithTimeout(context.Background(), s.ShutdownTimeout)
			defer ctxCancel()
		}

		err := s.Impl.Shutdown(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("shutdown: %w", err)
		}

		return err
	}

	serverDoneChan := make(chan error, 1)
	go func() {
		serverDoneChan <- s.Impl.ListenAndServe()
	}()

	var err error
	select {
	case err = <-serverDoneChan:
	case <-ctx.Done():
		err = shutdown()
	}
	if errors.Is(err, http.ErrServerClosed) || err == nil {
		return nil
	}

	return fmt.Errorf("http listener %s: %w", s.Impl.Addr, err)
}

func (s ServerImpl) Register(handler Handler, opts ...HandlerOption) {
	router := s.Impl.Handler.(*mux.Router)
	if len(opts) > 0 {
		router = router.NewRoute().Subrouter()
		for _, opt := range opts {
			opt(router)
		}
	}

	httpHandler := httpHandlerWrapper(handler.Handle)
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
