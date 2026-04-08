package user

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	"github.com/klwxsrx/go-service-template/internal/user/api"
	"github.com/klwxsrx/go-service-template/internal/user/app/encoding"
	"github.com/klwxsrx/go-service-template/internal/user/app/message"
	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	userappservicegenerated "github.com/klwxsrx/go-service-template/internal/user/app/service/generated"
	"github.com/klwxsrx/go-service-template/internal/user/app/session"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	"github.com/klwxsrx/go-service-template/internal/user/infra"
	"github.com/klwxsrx/go-service-template/internal/user/infra/http"
	userinfrahttpgenerated "github.com/klwxsrx/go-service-template/internal/user/infra/http/generated"
	"github.com/klwxsrx/go-service-template/internal/user/infra/password"
	userinfrasession "github.com/klwxsrx/go-service-template/internal/user/infra/session"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
	"github.com/klwxsrx/go-service-template/pkg/event"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/lazy"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/sql"
)

type (
	Container struct {
		API      APIContainer
		Handlers HandlerContainer
	}

	APIContainer struct {
		Auth lazy.Loader[api.AuthenticationService]
		User lazy.Loader[api.UserService]
	}

	HandlerContainer struct {
		authenticate         lazy.Loader[http.AuthenticateHandler]
		verifyAuthentication lazy.Loader[http.VerifyAuthenticationHandler]
		registerUser         lazy.Loader[http.RegisterUserHandler]
		getCurrentUser       lazy.Loader[http.GetCurrentUserHandler]
		getUserByID          lazy.Loader[http.GetUserByIDHandler]
		deleteUserByID       lazy.Loader[http.DeleteUserByIDHandler]
	}
)

func NewContainer(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[cmd.SQLMigrations],
	messagingEventDispatcher lazy.Loader[pkgmessage.EventDispatcher],
) *Container {
	eventDispatcher := lazy.New(eventDispatcherProvider(messagingEventDispatcher))

	transaction := lazy.New(transactionProvider(db))
	sqlContainer := lazy.New(infra.NewSQLContainer(db, dbMigrations, eventDispatcher))

	dtoConverter := lazy.New(dtoConverterProvider)
	passwordEncoder := lazy.New(passwordEncoderProvider)
	sessionTokenGenerator := lazy.New(sessionTokenGeneratorProvider)
	permissionService := lazy.New(permissionServiceProvider)

	authService := lazy.New(authServiceProvider(sessionTokenGenerator, passwordEncoder, sqlContainer))
	userService := lazy.New(userServiceProvider(passwordEncoder, permissionService, transaction, sqlContainer, dtoConverter))

	httpDTOConverter := lazy.New(httpDTOConverterProvider)
	return &Container{
		API: APIContainer{
			Auth: lazy.New(func() (api.AuthenticationService, error) {
				return authService.Load()
			}),
			User: lazy.New(func() (api.UserService, error) {
				return userService.Load()
			}),
		},
		Handlers: HandlerContainer{
			authenticate: lazy.New(func() (http.AuthenticateHandler, error) {
				return http.NewAuthenticateHandler(authService.MustLoad()), nil
			}),
			verifyAuthentication: lazy.New(func() (http.VerifyAuthenticationHandler, error) {
				return http.NewVerifyAuthenticationHandler(authService.MustLoad()), nil
			}),
			registerUser: lazy.New(func() (http.RegisterUserHandler, error) {
				return http.NewRegisterUserHandler(userService.MustLoad(), httpDTOConverter.MustLoad()), nil
			}),
			getCurrentUser: lazy.New(func() (http.GetCurrentUserHandler, error) {
				return http.NewGetCurrentUserHandler(userService.MustLoad(), httpDTOConverter.MustLoad()), nil
			}),
			getUserByID: lazy.New(func() (http.GetUserByIDHandler, error) {
				return http.NewGetUserByIDHandler(userService.MustLoad(), httpDTOConverter.MustLoad()), nil
			}),
			deleteUserByID: lazy.New(func() (http.DeleteUserByIDHandler, error) {
				return http.NewDeleteUserByIDHandler(userService.MustLoad()), nil
			}),
		},
	}
}

func (c *HandlerContainer) MustRegister(registry pkghttp.HandlerRegistry) {
	registry.Register(c.authenticate.MustLoad())
	registry.Register(c.verifyAuthentication.MustLoad())
	registry.Register(c.registerUser.MustLoad())
	registry.Register(c.getCurrentUser.MustLoad(), pkghttp.WithAuthenticationRequirement())
	registry.Register(c.getUserByID.MustLoad(), pkghttp.WithAuthenticationRequirement())
	registry.Register(c.deleteUserByID.MustLoad(), pkghttp.WithAuthenticationRequirement())
}

func eventDispatcherProvider(messagingEventDispatcher lazy.Loader[pkgmessage.EventDispatcher]) func() (event.Dispatcher, error) {
	return func() (event.Dispatcher, error) {
		eventDispatcher := messagingEventDispatcher.MustLoad()
		err := eventDispatcher.Register(pkgmessage.TopicMessages{
			message.TopicDomainEventUser: {
				pkgmessage.RegisterEvent[domain.EventUserDeleted](),
			},
		})
		if err != nil {
			panic(fmt.Errorf("register %s messages: %w", domain.Name, err))
		}

		return eventDispatcher, nil
	}
}

func transactionProvider(db lazy.Loader[sql.Database]) func() (persistence.Transaction, error) {
	return func() (persistence.Transaction, error) {
		return sql.NewTransaction(
			db.MustLoad(),
			domain.Name,
			nil,
		), nil
	}
}

func dtoConverterProvider() (service.DTOConverter, error) {
	return &userappservicegenerated.DTOConverterImpl{}, nil
}

func passwordEncoderProvider() (encoding.PasswordEncoder, error) {
	return password.NewEncoder(), nil
}

func sessionTokenGeneratorProvider() (session.TokenGenerator, error) {
	return userinfrasession.NewTokenGenerator(), nil
}

func permissionServiceProvider() (auth.PermissionService, error) {
	return pkgauth.NewPermissionService[auth.Principal](), nil
}

func authServiceProvider(
	sessionTokens lazy.Loader[session.TokenGenerator],
	passwordEncoder lazy.Loader[encoding.PasswordEncoder],
	sqlContainer lazy.Loader[infra.SQLContainer],
) func() (service.Authentication, error) {
	return func() (service.Authentication, error) {
		return service.NewAuthentication(
			sqlContainer.MustLoad().UserRepo.MustLoad(),
			sessionTokens.MustLoad(),
			passwordEncoder.MustLoad(),
		), nil
	}
}

func userServiceProvider(
	passwordEncoder lazy.Loader[encoding.PasswordEncoder],
	permissionService lazy.Loader[auth.PermissionService],
	transaction lazy.Loader[persistence.Transaction],
	sqlContainer lazy.Loader[infra.SQLContainer],
	dtoConverter lazy.Loader[service.DTOConverter],
) func() (service.User, error) {
	return func() (service.User, error) {
		return service.NewUser(
			sqlContainer.MustLoad().UserRepo.MustLoad(),
			passwordEncoder.MustLoad(),
			permissionService.MustLoad(),
			transaction.MustLoad(),
			dtoConverter.MustLoad(),
		), nil
	}
}

func httpDTOConverterProvider() (http.DTOConverter, error) {
	return &userinfrahttpgenerated.DTOConverterImpl{}, nil
}
