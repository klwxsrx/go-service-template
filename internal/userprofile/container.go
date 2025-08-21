package userprofile

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/pkg/cmd"
	userapi "github.com/klwxsrx/go-service-template/internal/user/api"
	userinfrahttp "github.com/klwxsrx/go-service-template/internal/user/infra/http"
	"github.com/klwxsrx/go-service-template/internal/userprofile/api"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/message"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/service"
	userprofileappservicegenerated "github.com/klwxsrx/go-service-template/internal/userprofile/app/service/generated"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/user"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	"github.com/klwxsrx/go-service-template/internal/userprofile/infra"
	"github.com/klwxsrx/go-service-template/internal/userprofile/infra/http"
	userprofileinfrahttpgenerated "github.com/klwxsrx/go-service-template/internal/userprofile/infra/http/generated"
	userprofileinfrauserhttp "github.com/klwxsrx/go-service-template/internal/userprofile/infra/user/http"
	userprofileinfrauserhttpgenerated "github.com/klwxsrx/go-service-template/internal/userprofile/infra/user/http/generated"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/idk"
	"github.com/klwxsrx/go-service-template/pkg/lazy"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/sql"
)

type DependencyContainer struct {
	UserProfileService lazy.Loader[api.UserProfileService]

	getUserProfileHandler    lazy.Loader[http.GetUserProfileHandler]
	updateUserProfileHandler lazy.Loader[http.UpdateUserProfileHandler]
}

func NewDependencyContainer(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[cmd.SQLMigrations],
	httpClientFactory lazy.Loader[cmd.HTTPClientFactory],
	idkService lazy.Loader[idk.Service],
) DependencyContainer {
	transaction := transactionProvider(db)
	sqlContainer := infra.NewSQLContainer(db, dbMigrations)

	dtoConverter := dtoConverterProvider()
	permissionService := permissionServiceProvider()

	userHTTPDTOConverter := userHTTPDTOConverterProvider()
	userService := userServiceProvider(httpClientFactory, userHTTPDTOConverter)

	userProfileService := userProfileServiceProvider(
		userService,
		permissionService,
		idkService,
		sqlContainer,
		transaction,
		dtoConverter,
	)

	httpDTOConverter := httpDTOConverterProvider()
	return DependencyContainer{
		UserProfileService: lazy.New(func() (api.UserProfileService, error) {
			return userProfileService.Load()
		}),
		getUserProfileHandler: lazy.New(func() (http.GetUserProfileHandler, error) {
			return http.NewGetUserProfileHandler(userProfileService.MustLoad(), httpDTOConverter.MustLoad()), nil
		}),
		updateUserProfileHandler: lazy.New(func() (http.UpdateUserProfileHandler, error) {
			return http.NewUpdateUserProfileHandler(userProfileService.MustLoad()), nil
		}),
	}
}

func (c *DependencyContainer) MustRegisterHTTPHandlers(registry pkghttp.HandlerRegistry) {
	options := []pkghttp.ServerOption{
		pkghttp.WithAuthenticationRequirement(),
	}

	registry.Register(c.getUserProfileHandler.MustLoad(), options...)
	registry.Register(c.updateUserProfileHandler.MustLoad(), options...)
}

func (c *DependencyContainer) MustRegisterMessageHandlers(registry pkgmessage.HandlerRegistry) {
	err := registry.RegisterHandlers(message.SubscriberName, pkgmessage.TopicHandlers{
		userapi.TopicDomainEventUser: {
			pkgmessage.RegisterEventHandlers[user.EventUserDeleted](c.UserProfileService.MustLoad().HandleUserDeleted),
		},
	})
	if err != nil {
		panic(fmt.Errorf("register %s message handlers: %w", domain.Name, err))
	}
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
		return &userprofileappservicegenerated.DTOConverterImpl{}, nil
	})
}

func httpDTOConverterProvider() lazy.Loader[http.DTOConverter] {
	return lazy.New(func() (http.DTOConverter, error) {
		return &userprofileinfrahttpgenerated.DTOConverterImpl{}, nil
	})
}

func permissionServiceProvider() lazy.Loader[auth.PermissionService] {
	return lazy.New(func() (auth.PermissionService, error) {
		return pkgauth.NewPermissionService[auth.Principal](), nil
	})
}

func userHTTPDTOConverterProvider() lazy.Loader[userprofileinfrauserhttp.DTOConverter] {
	return lazy.New(func() (userprofileinfrauserhttp.DTOConverter, error) {
		return &userprofileinfrauserhttpgenerated.DTOConverterImpl{}, nil
	})
}

func userServiceProvider(
	httpClientFactory lazy.Loader[cmd.HTTPClientFactory],
	dtoConverter lazy.Loader[userprofileinfrauserhttp.DTOConverter],
) lazy.Loader[user.Service] {
	return lazy.New(func() (user.Service, error) {
		httpClient := httpClientFactory.MustLoad().MustInitClient(userinfrahttp.Destination)
		return userprofileinfrauserhttp.NewUserService(httpClient, dtoConverter.MustLoad()), nil
	})
}

func userProfileServiceProvider(
	userService lazy.Loader[user.Service],
	permissionService lazy.Loader[auth.PermissionService],
	idkService lazy.Loader[idk.Service],
	sqlContainer lazy.Loader[infra.SQLContainer],
	transaction lazy.Loader[persistence.Transaction],
	dtoConverter lazy.Loader[service.DTOConverter],
) lazy.Loader[service.UserProfile] {
	return lazy.New(func() (service.UserProfile, error) {
		return service.NewUserProfile(
			userService.MustLoad(),
			sqlContainer.MustLoad().UserProfileRepo.MustLoad(),
			permissionService.MustLoad(),
			idkService.MustLoad(),
			transaction.MustLoad(),
			dtoConverter.MustLoad(),
		), nil
	})
}
