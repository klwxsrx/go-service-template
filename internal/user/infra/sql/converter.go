//go:generate go tool goverter gen -g wrapErrors -g "output:file ./generated/goverter_${GOFILE}" .
package sql

import "github.com/klwxsrx/go-service-template/internal/user/domain"

// goverter:converter
// goverter:skipCopySameType
type SqlxConverter interface {
	ToDomainUser(*SqlxUser) *domain.User
	ToDomainUsers([]SqlxUser) []domain.User
	// goverter:ignore Changes
	ToDomainUserValue(SqlxUser) domain.User
}
