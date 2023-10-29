package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/duck/app/external"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type DuckService struct {
	gooseService external.GooseService
	tx           persistence.Transaction
	duckRepo     domain.DuckRepo
}

func (s *DuckService) Create(ctx context.Context, name string) error {
	return s.tx.Execute(ctx, func(ctx context.Context) error {
		duck := domain.NewDuck(uuid.New(), strings.TrimSpace(name))
		err := s.duckRepo.Store(ctx, duck)
		if err != nil {
			return fmt.Errorf("failed to store duck, repo error: %w", err)
		}

		return nil
	})
}

func (s *DuckService) HandleDuckCreated(_ context.Context, _ domain.EventDuckCreated) error {
	err := s.gooseService.DoSome()
	if err != nil {
		return fmt.Errorf("failed to do some, goose error: %w", err)
	}

	return nil
}

func (s *DuckService) HandleGooseQuacked(_ context.Context, _ external.EventGooseQuacked) error {
	return nil
}

func NewDuckService(
	gooseService external.GooseService,
	tx persistence.Transaction,
	duckRepo domain.DuckRepo,
) *DuckService {
	return &DuckService{
		gooseService: gooseService,
		tx:           tx,
		duckRepo:     duckRepo,
	}
}
