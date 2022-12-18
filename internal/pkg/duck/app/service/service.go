package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type DuckService struct {
	duckRepo domain.DuckRepo
	tx       persistence.Transaction
}

func (s *DuckService) Create(ctx context.Context) error {
	return s.tx.Execute(ctx, func(ctx context.Context) error {
		duck := domain.NewDuck(uuid.New())

		err := s.duckRepo.Store(ctx, duck)
		if err != nil {
			return fmt.Errorf("failed to create duck, repo error: %w", err)
		}

		return nil
	})
}

func NewDuckService(
	duckRepo domain.DuckRepo,
	tx persistence.Transaction,
) *DuckService {
	return &DuckService{
		duckRepo: duckRepo,
		tx:       tx,
	}
}
