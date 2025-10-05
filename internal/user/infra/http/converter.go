//go:generate go tool goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package http

import (
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/user/app/service"
	"github.com/klwxsrx/go-service-template/internal/user/domain"
)

// goverter:converter
// goverter:skipCopySameType
// goverter:extend UserIDToUUID
type DTOConverter interface {
	ToHTTPUserOut(*service.UserData) *UserOut
	ToDTOUserCredentials(RegisterUserIn) service.UserCredentials
}

func UserIDToUUID(id domain.UserID) uuid.UUID {
	return id.UUID
}
