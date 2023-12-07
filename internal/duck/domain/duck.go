//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "DuckRepo=DuckRepo"
package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/event"
)

var ErrDuckNotFound = errors.New("duck not found")

type (
	DuckSpec struct {
		ID *uuid.UUID
	}

	DuckRepo interface {
		FindOne(ctx context.Context, spec DuckSpec) (*Duck, error)
		Store(ctx context.Context, duck *Duck) error
	}
)

type Duck struct {
	ID       uuid.UUID
	Name     string
	IsActive bool
	Changes  []event.Event
}

func NewDuck(
	id uuid.UUID,
	name string,
) *Duck {
	return &Duck{
		ID:       id,
		Name:     name,
		IsActive: true,
		Changes: []event.Event{EventDuckCreated{
			EventID: uuid.New(),
			DuckID:  id,
		}},
	}
}
