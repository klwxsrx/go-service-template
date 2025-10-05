//go:generate go tool goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package service

import "github.com/klwxsrx/go-service-template/internal/user/domain"

// goverter:converter
// goverter:skipCopySameType
type DTOConverter interface {
	ToDTOUserData(*domain.User) *UserData
	ToDTOUsersData([]domain.User) []UserData
}
