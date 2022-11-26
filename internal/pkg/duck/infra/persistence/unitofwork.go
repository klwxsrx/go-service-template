package persistence

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/app/service"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/domain"
	"github.com/klwxsrx/go-service-template/internal/pkg/duck/infra/sql"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

type unitOfWork struct {
	base            pkgsql.UnitOfWork
	eventSerializer message.EventSerializer
	onComplete      func()
}

func (u *unitOfWork) Execute(ctx context.Context, f func(ctx context.Context, tx service.Transaction) error) error {
	return u.base.Execute(ctx, func(ctx context.Context, dbTx pkgsql.ClientTx) error {
		return f(ctx, &tx{
			duckRepo:   duckRepo(dbTx, u.eventSerializer),
			db:         dbTx,
			onComplete: u.onComplete,
		})
	})
}

type tx struct {
	duckRepo   domain.DuckRepo
	db         pkgsql.ClientTx
	onComplete func()
}

func (t *tx) DuckRepo() domain.DuckRepo {
	return t.duckRepo
}

func (t *tx) Complete() error {
	err := t.db.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit db transaction: %w", err)
	}

	t.onComplete()
	return nil
}

func NewUnitOfWork(db pkgsql.TxClient, eventSerializer message.EventSerializer, onComplete func()) service.UnitOfWork {
	return &unitOfWork{
		base:            pkgsql.NewUnitOfWork(db),
		eventSerializer: eventSerializer,
		onComplete:      onComplete,
	}
}

func duckRepo(db pkgsql.Client, eventSerializer message.EventSerializer) domain.DuckRepo {
	return sql.NewDuckRepo(db, eventDispatcher(db, eventSerializer))
}

func eventDispatcher(db pkgsql.Client, eventSerializer message.EventSerializer) event.Dispatcher {
	store := messageStore(db)
	return message.NewEventDispatcher(
		persistence.NewMessageSender(store),
		eventSerializer,
	)
}

func messageStore(db pkgsql.Client) persistence.MessageStore {
	return pkgsql.NewMessageStore(db)
}
