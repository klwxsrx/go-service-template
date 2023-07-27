//go:generate ${PROJECT_PATH}/tools/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "DuckRepo=DuckRepo"
package domain

import (
	"context"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/event"
)

type DuckRepo interface {
	Store(ctx context.Context, duck *Duck) error
}

type Duck struct {
	ID      uuid.UUID
	Name    string
	Changes []event.Event
}

func NewDuck(
	id uuid.UUID,
	name string,
) *Duck {
	return &Duck{
		ID:   id,
		Name: name,
		Changes: []event.Event{EventDuckCreated{
			EventID: uuid.New(),
			DuckID:  id,
		}},
	}
}
