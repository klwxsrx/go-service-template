package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/internal/duck/api"
	"github.com/klwxsrx/go-service-template/internal/duck/app/goose"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

type DuckService struct {
	gooseService goose.Service
	transaction  persistence.Transaction
	duckRepo     domain.DuckRepo
}

func NewDuckService(
	gooseService goose.Service,
	transaction persistence.Transaction,
	duckRepo domain.DuckRepo,
) *DuckService {
	return &DuckService{
		gooseService: gooseService,
		transaction:  transaction,
		duckRepo:     duckRepo,
	}
}

func (s *DuckService) Create(ctx context.Context, name string) (uuid.UUID, error) {
	duckID := uuid.New()
	err := s.transaction.WithinContext(ctx, func(ctx context.Context) error {
		duck := domain.NewDuck(duckID, strings.TrimSpace(name))
		err := s.duckRepo.Store(ctx, duck)
		if err != nil {
			return fmt.Errorf("store duck: %w", err)
		}

		return nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}

	return duckID, nil
}

func (s *DuckService) SetActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	return s.transaction.WithinContext(ctx, func(ctx context.Context) error {
		duck, err := s.duckRepo.FindOne(
			s.transaction.WithLock(ctx),
			domain.DuckSpec{ID: &id},
		)
		if errors.Is(err, domain.ErrDuckNotFound) {
			return api.ErrDuckNotFound
		}
		if err != nil {
			return fmt.Errorf("get duck by id: %w", err)
		}

		if duck.IsActive == isActive {
			return nil
		}

		duck.IsActive = isActive
		err = s.duckRepo.Store(ctx, duck)
		if err != nil {
			return fmt.Errorf("store duck: %w", err)
		}

		return nil
	})
}

func (s *DuckService) HandleDuckCreated(_ context.Context, _ domain.EventDuckCreated) error {
	err := s.gooseService.DoSome()
	if err != nil {
		return fmt.Errorf("do some: %w", err)
	}

	return nil
}

func (s *DuckService) HandleGooseQuacked(_ context.Context, _ goose.EventGooseQuacked) error {
	return nil
}
