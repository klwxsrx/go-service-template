//go:generate ${TOOLS_BIN}/goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package http

import (
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/userprofile/app/service"
	"github.com/klwxsrx/go-service-template/internal/userprofile/domain"
)

// goverter:converter
// goverter:skipCopySameType
// goverter:extend UserIDToUUID
type DTOConverter interface {
	ToHTTPUserProfileOut(*service.UserProfileData) *UserProfileOut
}

func UserIDToUUID(id domain.UserID) uuid.UUID {
	return id.UUID
}
