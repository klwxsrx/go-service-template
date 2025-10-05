//go:generate go tool goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package http

import "github.com/klwxsrx/go-service-template/internal/userprofile/app/user"

// goverter:converter
// goverter:skipCopySameType
type DTOConverter interface {
	ToDTOUserData(*UserOut) *user.Data
}
