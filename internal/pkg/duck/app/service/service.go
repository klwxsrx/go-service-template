package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"strings"
)

type DuckService interface {
	Create(ctx context.Context, name string) error
	HandleDuckCreated(ctx context.Context, event domain.EventDuckCreated) error
	HandleGooseQuacked(ctx context.Context, event external.EventGooseQuacked) error
}

type duckService struct {
	duckRepo domain.DuckRepo
	tx       persistence.Transaction
}

func (s *duckService) Create(ctx context.Context, name string) error {
	return s.tx.Execute(ctx, func(ctx context.Context) error {
		duck := domain.NewDuck(uuid.New(), strings.TrimSpace(name))
		err := s.duckRepo.Store(ctx, duck)
		if err != nil {
			return fmt.Errorf("failed to store duck, repo error: %w", err)
		}

		return nil
	})
}

func (s *duckService) HandleDuckCreated(_ context.Context, _ domain.EventDuckCreated) error {
	return nil
}

func (s *duckService) HandleGooseQuacked(_ context.Context, _ external.EventGooseQuacked) error {
	return nil
}

func NewDuckService(
	duckRepo domain.DuckRepo,
	tx persistence.Transaction,
) DuckService {
	return &duckService{
		duckRepo: duckRepo,
		tx:       tx,
	}
}
