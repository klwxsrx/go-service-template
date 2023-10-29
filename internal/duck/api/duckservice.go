package api

import "context"

type DuckService interface {
	Create(ctx context.Context, name string) error
}
