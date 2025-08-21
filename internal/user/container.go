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

type DependencyContainer struct {
	AuthService lazy.Loader[api.AuthenticationService]
	UserService lazy.Loader[api.UserService]

	authenticateHandler         lazy.Loader[http.AuthenticateHandler]
	verifyAuthenticationHandler lazy.Loader[http.VerifyAuthenticationHandler]
	registerUserHandler         lazy.Loader[http.RegisterUserHandler]
	getCurrentUserHandler       lazy.Loader[http.GetCurrentUserHandler]
	getUserByIDHandler          lazy.Loader[http.GetUserByIDHandler]
	deleteUserByIDHandler       lazy.Loader[http.DeleteUserByIDHandler]
}

func NewDependencyContainer(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[cmd.SQLMigrations],
	messagingEventDispatcher lazy.Loader[pkgmessage.EventDispatcher],
) DependencyContainer {
	eventDispatcher := eventDispatcherProvider(messagingEventDispatcher)

	transaction := transactionProvider(db)
	sqlContainer := infra.NewSQLContainer(db, dbMigrations, eventDispatcher)

	dtoConverter := dtoConverterProvider()
	passwordEncoder := passwordEncoderProvider()
	sessionTokenGenerator := sessionTokenGeneratorProvider()
	permissionService := permissionServiceProvider()

	authService := authServiceProvider(sessionTokenGenerator, passwordEncoder, sqlContainer)
	userService := userServiceProvider(passwordEncoder, permissionService, transaction, sqlContainer, dtoConverter)

	httpDTOConverter := httpDTOConverterProvider()
	return DependencyContainer{
		AuthService: lazy.New(func() (api.AuthenticationService, error) {
			return authService.Load()
		}),
		UserService: lazy.New(func() (api.UserService, error) {
			return userService.Load()
		}),
		authenticateHandler: lazy.New(func() (http.AuthenticateHandler, error) {
			return http.NewAuthenticateHandler(authService.MustLoad()), nil
		}),
		verifyAuthenticationHandler: lazy.New(func() (http.VerifyAuthenticationHandler, error) {
			return http.NewVerifyAuthenticationHandler(authService.MustLoad()), nil
		}),
		registerUserHandler: lazy.New(func() (http.RegisterUserHandler, error) {
			return http.NewRegisterUserHandler(userService.MustLoad(), httpDTOConverter.MustLoad()), nil
		}),
		getCurrentUserHandler: lazy.New(func() (http.GetCurrentUserHandler, error) {
			return http.NewGetCurrentUserHandler(userService.MustLoad(), httpDTOConverter.MustLoad()), nil
		}),
		getUserByIDHandler: lazy.New(func() (http.GetUserByIDHandler, error) {
			return http.NewGetUserByIDHandler(userService.MustLoad(), httpDTOConverter.MustLoad()), nil
		}),
		deleteUserByIDHandler: lazy.New(func() (http.DeleteUserByIDHandler, error) {
			return http.NewDeleteUserByIDHandler(userService.MustLoad()), nil
		}),
	}
}

func (c *DependencyContainer) MustRegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	registry.Register(c.authenticateHandler.MustLoad())
	registry.Register(c.verifyAuthenticationHandler.MustLoad())
	registry.Register(c.registerUserHandler.MustLoad())
	registry.Register(c.getCurrentUserHandler.MustLoad(), pkghttp.WithAuthenticationRequirement())
	registry.Register(c.getUserByIDHandler.MustLoad(), pkghttp.WithAuthenticationRequirement())
	registry.Register(c.deleteUserByIDHandler.MustLoad(), pkghttp.WithAuthenticationRequirement())
}

func eventDispatcherProvider(messagingEventDispatcher lazy.Loader[pkgmessage.EventDispatcher]) lazy.Loader[event.Dispatcher] {
	return lazy.New(func() (event.Dispatcher, error) {
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
	})
}

func transactionProvider(db lazy.Loader[sql.Database]) lazy.Loader[persistence.Transaction] {
	return lazy.New(func() (persistence.Transaction, error) {
		return sql.NewTransaction(
			db.MustLoad(),
			domain.Name,
			nil,
		), nil
	})
}

func dtoConverterProvider() lazy.Loader[service.DTOConverter] {
	return lazy.New(func() (service.DTOConverter, error) {
		return &userappservicegenerated.DTOConverterImpl{}, nil
	})
}

func passwordEncoderProvider() lazy.Loader[encoding.PasswordEncoder] {
	return lazy.New(func() (encoding.PasswordEncoder, error) {
		return password.NewEncoder(), nil
	})
}

func sessionTokenGeneratorProvider() lazy.Loader[session.TokenGenerator] {
	return lazy.New(func() (session.TokenGenerator, error) {
		return userinfrasession.NewTokenGenerator(), nil
	})
}

func permissionServiceProvider() lazy.Loader[auth.PermissionService] {
	return lazy.New(func() (auth.PermissionService, error) {
		return pkgauth.NewPermissionService[auth.Principal](), nil
	})
}

func authServiceProvider(
	sessionTokens lazy.Loader[session.TokenGenerator],
	passwordEncoder lazy.Loader[encoding.PasswordEncoder],
	sqlContainer lazy.Loader[infra.SQLContainer],
) lazy.Loader[service.Authentication] {
	return lazy.New(func() (service.Authentication, error) {
		return service.NewAuthentication(
			sqlContainer.MustLoad().UserRepo.MustLoad(),
			sessionTokens.MustLoad(),
			passwordEncoder.MustLoad(),
		), nil
	})
}

func userServiceProvider(
	passwordEncoder lazy.Loader[encoding.PasswordEncoder],
	permissionService lazy.Loader[auth.PermissionService],
	transaction lazy.Loader[persistence.Transaction],
	sqlContainer lazy.Loader[infra.SQLContainer],
	dtoConverter lazy.Loader[service.DTOConverter],
) lazy.Loader[service.User] {
	return lazy.New(func() (service.User, error) {
		return service.NewUser(
			sqlContainer.MustLoad().UserRepo.MustLoad(),
			passwordEncoder.MustLoad(),
			permissionService.MustLoad(),
			transaction.MustLoad(),
			dtoConverter.MustLoad(),
		), nil
	})
}

func httpDTOConverterProvider() lazy.Loader[http.DTOConverter] {
	return lazy.New(func() (http.DTOConverter, error) {
		return &userinfrahttpgenerated.DTOConverterImpl{}, nil
	})
}
