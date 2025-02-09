//go:generate ${TOOLS_BIN}/goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package sql

import "github.com/klwxsrx/go-service-template/internal/userprofile/domain"

// goverter:converter
// goverter:skipCopySameType
type SqlxConverter interface {
	ToDomainUserProfile(*SqlxUserProfile) *domain.UserProfile
}
