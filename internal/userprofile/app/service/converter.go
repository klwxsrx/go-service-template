//go:generate go tool goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package service

import "github.com/klwxsrx/go-service-template/internal/userprofile/domain"

// goverter:converter
// goverter:skipCopySameType
type DTOConverter interface {
	ToDTOUserProfileData(*domain.UserProfile) *UserProfileData
}
