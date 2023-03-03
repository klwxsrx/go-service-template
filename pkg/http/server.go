package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"net/http"
	"os"
	"time"
)

const (
	DefaultServerAddress = ":8080"

	defaultReadTimeout       = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
)

type Option func(router *mux.Router)
type Middleware func(http.Handler) http.Handler

type Handler interface {
	Method() string
	Path() string
	HTTPHandler() http.HandlerFunc
}

func Must(err error) {
	if err != nil {
		panic(fmt.Errorf("unable to listen the server: %w", err))
	}
}

type Server interface {
	ListenAndServe(ctx context.Context, termSignalsChan <-chan os.Signal) error
	ListenAndServeProcess(ctx context.Context) hub.Process
	Register(handler Handler, opts ...Option)
}

type server struct {
	srv    *http.Server
	router *mux.Router
}

type serverProcess struct {
	ctx context.Context
	srv *http.Server
}

func (p *serverProcess) Name() string {
	return fmt.Sprintf("http server %s", p.srv.Addr)
}

func (p *serverProcess) Func() func(stopChan <-chan struct{}) error {
	return func(stopChan <-chan struct{}) error {
		return listenAndServe(p.ctx, p.srv, stopChan)
	}
}

func (s *server) ListenAndServeProcess(ctx context.Context) hub.Process {
	return &serverProcess{ctx, s.srv}
}

func (s *server) ListenAndServe(ctx context.Context, termSignalsChan <-chan os.Signal) error {
	return listenAndServe(ctx, s.srv, termSignalsChan)
}

func (s *server) Register(handler Handler, opts ...Option) {
	router := s.router
	if len(opts) > 0 {
		router = s.router.NewRoute().Subrouter()
		for _, opt := range opts {
			opt(router)
		}
	}

	router.
		Name(getRouteName(handler.Method(), handler.Path())).
		Methods(handler.Method()).
		Path(handler.Path()).
		HandlerFunc(handler.HTTPHandler())
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
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}
	return nil
}

func NewServer(address string, opts ...Option) Server {
	router := mux.NewRouter()
	for _, opt := range opts {
		opt(router)
	}

	srv := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadTimeout:       defaultReadTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	return &server{
		srv:    srv,
		router: router,
	}
}
