package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
)

type Transaction interface {
	DuckRepo() domain.DuckRepo
	Complete() error
}

type UnitOfWork interface {
	Execute(context.Context, func(ctx context.Context, tx Transaction) error) error
}

type DuckService struct {
	ufw UnitOfWork
}

func (s *DuckService) Create(ctx context.Context) error {
	return s.ufw.Execute(ctx, func(ctx context.Context, tx Transaction) error {
		duck := domain.NewDuck(uuid.New())

		err := tx.DuckRepo().Store(ctx, duck)
		if err != nil {
			return fmt.Errorf("failed to create duck, repo error: %w", err)
		}

		return tx.Complete()
	})
}

func NewDuckService(ufw UnitOfWork) *DuckService {
	return &DuckService{ufw: ufw}
}
