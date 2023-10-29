package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/klwxsrx/go-service-template/internal/duck/app/external"
	duckappexternalmock "github.com/klwxsrx/go-service-template/internal/duck/app/external/mock"
	"github.com/klwxsrx/go-service-template/internal/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/duck/domain"
	duckdomainmock "github.com/klwxsrx/go-service-template/internal/duck/domain/mock"
	pkgpersistencemock "github.com/klwxsrx/go-service-template/pkg/persistence/mock"
)

func TestDuckService_Create_Returns(t *testing.T) {
	tests := []struct {
		name        string
		transaction func(ctrl *gomock.Controller) *pkgpersistencemock.Transaction
		duckRepo    func(ctrl *gomock.Controller) *duckdomainmock.DuckRepo
		expect      func(t *testing.T, err error)
	}{
		{
			name: "success",
			duckRepo: func(ctrl *gomock.Controller) *duckdomainmock.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().Store(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, duck *domain.Duck) {
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
			transaction: func(ctrl *gomock.Controller) *pkgpersistencemock.Transaction {
				testFunc := func(ctx context.Context, fn func(context.Context) error, _ ...string) error {
					return fn(ctx)
				}

				mock := pkgpersistencemock.NewTransaction(ctrl)
				mock.EXPECT().Execute(gomock.Any(), gomock.Any(), []string{}).DoAndReturn(testFunc)
				return mock
			},
			expect: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error_when_repo_returns_error",
			duckRepo: func(ctrl *gomock.Controller) *duckdomainmock.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().Store(gomock.Any(), gomock.Any()).Return(errors.New("unexpected"))
				return mock
			},
			transaction: func(ctrl *gomock.Controller) *pkgpersistencemock.Transaction {
				testFunc := func(ctx context.Context, fn func(context.Context) error, _ ...string) error {
					return fn(ctx)
				}

				mock := pkgpersistencemock.NewTransaction(ctrl)
				mock.EXPECT().Execute(gomock.Any(), gomock.Any(), []string{}).DoAndReturn(testFunc)
				return mock
			},
			expect: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "error_when_transaction_returns_error",
			duckRepo: func(ctrl *gomock.Controller) *duckdomainmock.DuckRepo {
				mock := duckdomainmock.NewDuckRepo(ctrl)
				mock.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)
				return mock
			},
			transaction: func(ctrl *gomock.Controller) *pkgpersistencemock.Transaction {
				testFunc := func(ctx context.Context, fn func(context.Context) error, _ ...string) error {
					_ = fn(ctx)
					return errors.New("unexpected")
				}

				mock := pkgpersistencemock.NewTransaction(ctrl)
				mock.EXPECT().Execute(gomock.Any(), gomock.Any(), []string{}).DoAndReturn(testFunc)
				return mock
			},
			expect: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := service.NewDuckService(
				duckappexternalmock.NewGooseService(ctrl),
				tc.transaction(ctrl),
				tc.duckRepo(ctrl),
			)

			duckName := "SomeDuckName"
			err := srv.Create(context.Background(), duckName)
			tc.expect(t, err)
		})
	}
}

func TestDuckService_HandleDuckCreated_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGooseService := duckappexternalmock.NewGooseService(ctrl)
	mockGooseService.EXPECT().DoSome().Return(nil)
	mockTransaction := pkgpersistencemock.NewTransaction(ctrl)
	mockDuckRepo := duckdomainmock.NewDuckRepo(ctrl)
	srv := service.NewDuckService(mockGooseService, mockTransaction, mockDuckRepo)

	err := srv.HandleDuckCreated(context.Background(), domain.EventDuckCreated{})
	assert.NoError(t, err)
}

func TestDuckService_HandleGooseQuacked_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockGooseService := duckappexternalmock.NewGooseService(ctrl)
	mockTransaction := pkgpersistencemock.NewTransaction(ctrl)
	mockDuckRepo := duckdomainmock.NewDuckRepo(ctrl)
	srv := service.NewDuckService(mockGooseService, mockTransaction, mockDuckRepo)

	err := srv.HandleGooseQuacked(context.Background(), external.EventGooseQuacked{})
	assert.NoError(t, err)
}