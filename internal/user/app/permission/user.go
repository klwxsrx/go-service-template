package permission

import (
	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
)

func CanReadUser(id domain.UserID) pkgauth.Permission[auth.Principal] {
	return func(auth pkgauth.Authentication[auth.Principal]) (bool, error) {
		if auth.Principal() == nil {
			return false, nil
		}

		switch {
		case auth.Principal().UserID != nil:
			return *auth.Principal().UserID == id.UUID, nil
		case auth.Principal().AdminUserID != nil:
			return true, nil
		case auth.Principal().ServiceName != nil:
			return true, nil
		default:
			return false, nil
		}
	}
}

func CanReadUserFilter(ids []domain.UserID) pkgauth.PermissionFilter[[]domain.UserID, auth.Principal] {
	return func(auth pkgauth.Authentication[auth.Principal]) ([]domain.UserID, error) {
		if auth.Principal() == nil {
			return nil, nil
		}

		switch {
		case auth.Principal().UserID != nil:
			userID := *auth.Principal().UserID
			for _, id := range ids {
				if id.UUID == userID {
					return []domain.UserID{id}, nil
				}
			}
			return nil, nil
		case auth.Principal().AdminUserID != nil:
			return ids, nil
		case auth.Principal().ServiceName != nil:
			return ids, nil
		default:
			return nil, nil
		}
	}
}

func CanDeleteUser(id domain.UserID) pkgauth.Permission[auth.Principal] {
	return func(auth pkgauth.Authentication[auth.Principal]) (bool, error) {
		if auth.Principal() == nil {
			return false, nil
		}

		switch {
		case auth.Principal().UserID != nil:
			return *auth.Principal().UserID == id.UUID, nil
		case auth.Principal().AdminUserID != nil:
			return true, nil
		default:
			return false, nil
		}
	}
}
