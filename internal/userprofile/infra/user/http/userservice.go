package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	internalhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/auth"
	"github.com/klwxsrx/go-service-template/internal/userprofile/app/user"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
)

var getUserByIDRoute = pkghttp.Route{Method: http.MethodGet, URL: "/users/{userID}"}

type userService struct {
	client    pkghttp.Client
	converter DTOConverter
}

func NewUserService(client pkghttp.Client, dtoConverter DTOConverter) user.Service {
	client = client.With(internalhttp.WithServiceNameAuth(auth.ServiceName))
	return userService{client: client, converter: dtoConverter}
}

func (s userService) Get(ctx context.Context, userID domain.UserID) (*user.Data, error) {
	resp, err := s.client.NewRequest(ctx, getUserByIDRoute).
		SetPathParam("userID", userID.String()).
		Send()
	if err != nil {
		return nil, fmt.Errorf("request user.getUserByID: %w", err)
	}
	defer resp.Close()

	if resp.StatusCode() == http.StatusNotFound {
		return nil, user.ErrUserNotFound
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("request user.getUserByID: invalid status code %d", resp.StatusCode())
	}
	body, err := pkghttp.ParseResponse(resp, pkghttp.JSONBody[UserOut](), nil)
	if err != nil {
		return nil, fmt.Errorf("user.getUserByID response: %w", err)
	}

	return s.converter.ToDTOUserData(&body), nil
}

type UserOut struct {
	ID        domain.UserID `json:"id"`
	DeletedAt *time.Time    `json:"deletedAt"`
}
