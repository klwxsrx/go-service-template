package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/klwxsrx/go-service-template/internal/duck/api"
	"github.com/klwxsrx/go-service-template/internal/duck/app/goose"
	duckappgoosemock "github.com/klwxsrx/go-service-template/internal/duck/app/goose/mock"
	"github.com/klwxsrx/go-service-template/internal/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	duckdomainmock "github.com/klwxsrx/go-service-template/internal/duck/domain/mock"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	pkgpersistencemock "github.com/klwxsrx/go-service-template/pkg/persistence/mock"
	pkgpersistencestub "github.com/klwxsrx/go-service-template/pkg/persistence/stub"
)

func TestDuckService_Create_Returns(t *testing.T) {
	tests := []struct {
		name        string
		transaction func(ctrl *gomock.Controller) persistence.Transaction
		duckRepo    func(ctrl *gomock.Controller) domain.DuckRepo
		expect      func(t *testing.T, err error)
	}{
		{
			name: "success",
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().Store(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, duck *domain.Duck) {
						assert.NotNil(t, duck)
						assert.NotEqual(t, duck.ID, uuid.Nil)
						assert.Len(t, duck.Changes, 1)
						assert.IsType(t, domain.EventDuckCreated{}, duck.Changes[0])

						evt := duck.Changes[0].(domain.EventDuckCreated)
						assert.Equal(t, duck.ID, evt.DuckID)
					}).
					Return(nil)
				return mock
			},
			expect: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error_when_repo_returns_error",
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().Store(gomock.Any(), gomock.Any()).Return(errors.New("unexpected"))
				return mock
			},
			expect: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "error_when_transaction_returns_error",
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				return duckdomainmock.NewDuckRepo(ctrl)
			},
			transaction: func(ctrl *gomock.Controller) persistence.Transaction {
				testFunc := func(_ context.Context, _ func(context.Context) error, _ ...string) error {
					return errors.New("unexpected")
				}

				mock := pkgpersistencemock.NewTransaction(ctrl)
				mock.EXPECT().WithinContext(gomock.Any(), gomock.Any(), []string{}).DoAndReturn(testFunc)
				return mock
			},
			expect: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			transactionMock := pkgpersistencestub.NewTransaction()
			if tc.transaction != nil {
				transactionMock = tc.transaction(ctrl)
			}

			srv := service.NewDuckService(
				duckappgoosemock.NewService(ctrl),
				transactionMock,
				tc.duckRepo(ctrl),
			)

			duckName := "SomeDuckName"
			_, err := srv.Create(context.Background(), duckName)
			tc.expect(t, err)
		})
	}
}

func TestDuckService_SetActive_Returns(t *testing.T) {
	duckID := uuid.New()
	duckName := "SomeDuckName"

	tests := []struct {
		name        string
		setIsActive bool
		transaction func(ctrl *gomock.Controller) persistence.Transaction
		duckRepo    func(ctrl *gomock.Controller) domain.DuckRepo
		expect      func(t *testing.T, err error)
	}{
		{
			name:        "success_when_deactivated",
			setIsActive: false,
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				duck := domain.Duck{
					ID:       duckID,
					Name:     duckName,
					IsActive: true,
					Changes:  nil,
				}

				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(&duck, nil)

				storedDuck := duck
				storedDuck.IsActive = false
				mock.EXPECT().Store(gomock.Any(), &storedDuck).Return(nil)

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:        "success_when_activated",
			setIsActive: true,
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				duck := domain.Duck{
					ID:       duckID,
					Name:     duckName,
					IsActive: false,
					Changes:  nil,
				}

				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(&duck, nil)

				storedDuck := duck
				storedDuck.IsActive = true
				mock.EXPECT().Store(gomock.Any(), &storedDuck).Return(nil)

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:        "success_when_repeatedly_deactivated",
			setIsActive: false,
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				duck := domain.Duck{
					ID:       duckID,
					Name:     duckName,
					IsActive: false,
					Changes:  nil,
				}

				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(&duck, nil)

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:        "success_when_repeatedly_activated",
			setIsActive: true,
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				duck := domain.Duck{
					ID:       duckID,
					Name:     duckName,
					IsActive: true,
					Changes:  nil,
				}

				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(&duck, nil)

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:        "error_when_repository_store_returns_error",
			setIsActive: false,
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				duck := domain.Duck{
					ID:       duckID,
					Name:     duckName,
					IsActive: true,
					Changes:  nil,
				}

				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(&duck, nil)
				mock.EXPECT().Store(gomock.Any(), gomock.Any()).Return(errors.New("unexpected"))

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "error_when_repository_find_returns_error",
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(nil, errors.New("unexpected"))

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "error_when_duck_not_found",
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().FindOne(gomock.Any(), domain.DuckSpec{ID: &duckID}).Return(nil, domain.ErrDuckNotFound)

				return mock
			},
			expect: func(t *testing.T, err error) {
				require.ErrorIs(t, err, api.ErrDuckNotFound)
			},
		},
		{
			name: "error_when_transaction_returns_error",
			transaction: func(ctrl *gomock.Controller) persistence.Transaction {
				testFunc := func(_ context.Context, _ func(context.Context) error, _ ...string) error {
					return errors.New("unexpected")
				}

				mock := pkgpersistencemock.NewTransaction(ctrl)
				mock.EXPECT().WithinContext(gomock.Any(), gomock.Any(), []string{}).DoAndReturn(testFunc)
				return mock
			},
			duckRepo: func(ctrl *gomock.Controller) domain.DuckRepo {
				return duckdomainmock.NewDuckRepo(ctrl)
			},
			expect: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			transactionMock := pkgpersistencestub.NewTransaction()
			if tc.transaction != nil {
				transactionMock = tc.transaction(ctrl)
			}

			srv := service.NewDuckService(
				duckappgoosemock.NewService(ctrl),
				transactionMock,
				tc.duckRepo(ctrl),
			)

			err := srv.SetActive(context.Background(), duckID, tc.setIsActive)
			tc.expect(t, err)
		})
	}
}

func TestDuckService_HandleDuckCreated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGooseService := duckappgoosemock.NewService(ctrl)
	mockGooseService.EXPECT().DoSome().Return(nil)
	mockTransaction := pkgpersistencemock.NewTransaction(ctrl)
	mockDuckRepo := duckdomainmock.NewDuckRepo(ctrl)
	srv := service.NewDuckService(mockGooseService, mockTransaction, mockDuckRepo)

	err := srv.HandleDuckCreated(context.Background(), domain.EventDuckCreated{})
	assert.NoError(t, err)
}

func TestDuckService_HandleDuckCreated_ErrorWhenGooseServiceReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGooseService := duckappgoosemock.NewService(ctrl)
	mockGooseService.EXPECT().DoSome().Return(errors.New("unexpected"))
	mockTransaction := pkgpersistencemock.NewTransaction(ctrl)
	mockDuckRepo := duckdomainmock.NewDuckRepo(ctrl)
	srv := service.NewDuckService(mockGooseService, mockTransaction, mockDuckRepo)

	err := srv.HandleDuckCreated(context.Background(), domain.EventDuckCreated{})
	assert.Error(t, err)
}

func TestDuckService_HandleGooseQuacked_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGooseService := duckappgoosemock.NewService(ctrl)
	mockTransaction := pkgpersistencemock.NewTransaction(ctrl)
	mockDuckRepo := duckdomainmock.NewDuckRepo(ctrl)
	srv := service.NewDuckService(mockGooseService, mockTransaction, mockDuckRepo)

	err := srv.HandleGooseQuacked(context.Background(), goose.EventGooseQuacked{})
	assert.NoError(t, err)
}
