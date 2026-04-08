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

type (
	Container struct {
		API      APIContainer
		Services ServiceContainer
		Handlers HandlerContainer
	}

	APIContainer struct {
		UserProfile lazy.Loader[api.UserProfileService]
	}

	ServiceContainer struct {
		userProfile lazy.Loader[service.UserProfile]
	}

	HandlerContainer struct {
		getUserProfile    lazy.Loader[http.GetUserProfileHandler]
		updateUserProfile lazy.Loader[http.UpdateUserProfileHandler]
	}
)

func NewContainer(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[cmd.SQLMigrations],
	httpClientFactory lazy.Loader[cmd.HTTPClientFactory],
	idkService lazy.Loader[idk.Service],
) *Container {
	transaction := lazy.New(transactionProvider(db))
	sqlContainer := lazy.New(infra.NewSQLContainer(db, dbMigrations))

	dtoConverter := lazy.New(dtoConverterProvider)
	permissionService := lazy.New(permissionServiceProvider)

	userHTTPDTOConverter := lazy.New(userHTTPDTOConverterProvider)
	userService := lazy.New(userServiceProvider(httpClientFactory, userHTTPDTOConverter))

	userProfileService := lazy.New(userProfileServiceProvider(
		userService,
		permissionService,
		idkService,
		sqlContainer,
		transaction,
		dtoConverter,
	))

	httpDTOConverter := lazy.New(httpDTOConverterProvider)
	return &Container{
		API: APIContainer{
			UserProfile: lazy.New(func() (api.UserProfileService, error) {
				return userProfileService.Load()
			}),
		},
		Services: ServiceContainer{
			userProfile: userProfileService,
		},
		Handlers: HandlerContainer{
			getUserProfile: lazy.New(func() (http.GetUserProfileHandler, error) {
				return http.NewGetUserProfileHandler(userProfileService.MustLoad(), httpDTOConverter.MustLoad()), nil
			}),
			updateUserProfile: lazy.New(func() (http.UpdateUserProfileHandler, error) {
				return http.NewUpdateUserProfileHandler(userProfileService.MustLoad()), nil
			}),
		},
	}
}

func (c *ServiceContainer) MustRegister(registry pkgmessage.HandlerRegistry) {
	err := registry.RegisterHandlers(message.SubscriberName, pkgmessage.TopicHandlers{
		userapi.TopicDomainEventUser: {
			pkgmessage.RegisterEventHandlers[user.EventUserDeleted](c.userProfile.MustLoad().HandleUserDeleted),
		},
	})
	if err != nil {
		panic(fmt.Errorf("register %s message handlers: %w", domain.Name, err))
	}
}

func (c *HandlerContainer) MustRegister(registry pkghttp.HandlerRegistry) {
	options := []pkghttp.HandlerOption{
		pkghttp.WithAuthenticationRequirement(),
	}

	registry.Register(c.getUserProfile.MustLoad(), options...)
	registry.Register(c.updateUserProfile.MustLoad(), options...)
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
	return &userprofileappservicegenerated.DTOConverterImpl{}, nil
}

func httpDTOConverterProvider() (http.DTOConverter, error) {
	return &userprofileinfrahttpgenerated.DTOConverterImpl{}, nil
}

func permissionServiceProvider() (auth.PermissionService, error) {
	return pkgauth.NewPermissionService[auth.Principal](), nil
}

func userHTTPDTOConverterProvider() (userprofileinfrauserhttp.DTOConverter, error) {
	return &userprofileinfrauserhttpgenerated.DTOConverterImpl{}, nil
}

func userServiceProvider(
	httpClientFactory lazy.Loader[cmd.HTTPClientFactory],
	dtoConverter lazy.Loader[userprofileinfrauserhttp.DTOConverter],
) func() (user.Service, error) {
	return func() (user.Service, error) {
		httpClient := httpClientFactory.MustLoad().MustInitClient(userinfrahttp.Destination)
		return userprofileinfrauserhttp.NewUserService(httpClient, dtoConverter.MustLoad()), nil
	}
}

func userProfileServiceProvider(
	userService lazy.Loader[user.Service],
	permissionService lazy.Loader[auth.PermissionService],
	idkService lazy.Loader[idk.Service],
	sqlContainer lazy.Loader[infra.SQLContainer],
	transaction lazy.Loader[persistence.Transaction],
	dtoConverter lazy.Loader[service.DTOConverter],
) func() (service.UserProfile, error) {
	return func() (service.UserProfile, error) {
		return service.NewUserProfile(
			userService.MustLoad(),
			sqlContainer.MustLoad().UserProfileRepo.MustLoad(),
			permissionService.MustLoad(),
			idkService.MustLoad(),
			transaction.MustLoad(),
			dtoConverter.MustLoad(),
		), nil
	}
}
