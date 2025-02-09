package permission

import (
	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
)

func CanGetUserProfile(id domain.UserID) pkgauth.Permission[auth.Principal] {
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

func CanUpdateUserProfile(id domain.UserID) pkgauth.Permission[auth.Principal] {
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
