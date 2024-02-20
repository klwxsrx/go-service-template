//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "API=API"
package api

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrDuckNotFound = errors.New("duck not found")

type API interface {
	Create(ctx context.Context, name string) (uuid.UUID, error)
	SetActive(ctx context.Context, id uuid.UUID, isActive bool) error
}
