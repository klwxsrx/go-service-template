//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "DuckService=DuckService"
package api

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrDuckNotFound = errors.New("duck not found")

type DuckService interface {
	Create(ctx context.Context, name string) (uuid.UUID, error)
	SetActive(ctx context.Context, id uuid.UUID, isActive bool) error
}
