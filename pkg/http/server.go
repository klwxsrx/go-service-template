package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/mux"

	"github.com/klwxsrx/go-service-template/pkg/worker"
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

func Must(err error) {
	if err != nil {
		panic(fmt.Errorf("listen the server: %w", err))
	}
}

type HandlerRegistry interface {
	Register(handler Handler, opts ...ServerOption)
}

type Server interface {
	Listen(ctx context.Context, termSignalsChan <-chan os.Signal) error
	NewListener(ctx context.Context) worker.NamedProcess
	HandlerRegistry
}

type server struct {
	srv    *http.Server
	router *mux.Router
}

type serverProcess struct {
	ctx context.Context
	srv *http.Server
}

func (p serverProcess) Name() string {
	return fmt.Sprintf("http server %s", p.srv.Addr)
}

func (p serverProcess) Process() worker.Process {
	return func(stopChan <-chan struct{}) error {
		return listenAndServe(p.ctx, p.srv, stopChan)
	}
}

func (s server) NewListener(ctx context.Context) worker.NamedProcess {
	return serverProcess{ctx, s.srv}
}

func (s server) Listen(ctx context.Context, termSignalsChan <-chan os.Signal) error {
	return listenAndServe(ctx, s.srv, termSignalsChan)
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

func listenAndServe[signal any](ctx context.Context, srv *http.Server, termSignal <-chan signal) error {
	serverDoneChan := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverDoneChan <- err
	}()

	var err error
	select {
	case err = <-serverDoneChan:
	case <-termSignal:
		err = shutdown(ctx, srv)
	}
	return err
}

func shutdown(ctx context.Context, srv *http.Server) error {
	err := srv.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}

func getRouteName(method, path string) string {
	path = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, strings.Trim(path, "/"))
	return strings.ToLower(fmt.Sprintf("%s_%s", method, path))
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
