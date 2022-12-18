package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"net/http"
	"sync"
	"time"
)

const DefaultServerAddress = ":8080"

type Option func(router *mux.Router)
type Middleware func(http.Handler) http.Handler

type Handler interface {
	Method() string
	Path() string
	HTTPHandler() http.HandlerFunc
}

type Server interface {
	MustListenAndServe()
	Register(handler Handler, opts ...Option)
	Shutdown(ctx context.Context)
}

type server struct {
	srv         *http.Server
	router      *mux.Router
	logger      log.Logger
	onceStarter *sync.Once
}

func (s *server) MustListenAndServe() {
	s.onceStarter.Do(func() {
		go func() {
			if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				panic(fmt.Errorf("unable to listen the server: %w", err))
			}
		}()
	})
}

func (s *server) Register(handler Handler, opts ...Option) {
	router := s.router.
		Name(getRouteName(handler.Method(), handler.Path())).
		Methods(handler.Method()).
		Path(handler.Path()).
		HandlerFunc(handler.HTTPHandler()).
		Subrouter()

	for _, opt := range opts {
		opt(router)
	}
}

func (s *server) Shutdown(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.WithError(err).Error(ctx, "failed to shutdown http server")
	}
}

func NewServer(address string, logger log.Logger, opts ...Option) Server {
	router := mux.NewRouter()
	for _, opt := range opts {
		opt(router)
	}

	srv := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &server{
		srv:         srv,
		router:      router,
		logger:      logger,
		onceStarter: &sync.Once{},
	}
}
