package sql

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/message"
)

const messageOutboxLockName = "message_outbox"

type messageOutboxStorage struct {
	db Client
}

func NewMessageOutboxStorage(db Client) message.OutboxStorage {
	return messageOutboxStorage{db: db}
}

func (s messageOutboxStorage) Lock(ctx context.Context) (release func() error, err error) {
	return withSessionLevelLock(ctx, messageOutboxLockName, s.db)
}

func (s messageOutboxStorage) GetBatch(ctx context.Context, scheduledBefore time.Time, batchSize int) ([]message.Message, error) {
	query, args, err := sq.
		Select("id", "topic", "key", "payload").
		From("message_outbox").
		Where(sq.LtOrEq{"scheduled_at": scheduledBefore}).
		OrderBy("scheduled_at").
		Limit(uint64(batchSize)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql: %w", err)
	}

	var sqlxResult []sqlxMessage
	err = s.db.SelectContext(ctx, &sqlxResult, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select messages: %w", err)
	}

	result := make([]message.Message, 0, len(sqlxResult))
	for _, sqlxMsg := range sqlxResult {
		result = append(result, message.Message{
			ID:      sqlxMsg.ID,
			Topic:   sqlxMsg.Topic,
			Key:     sqlxMsg.Key,
			Payload: sqlxMsg.Payload,
		})
	}
	return result, nil
}

func (s messageOutboxStorage) Store(ctx context.Context, msgs []message.Message, scheduledAt time.Time) error {
	if len(msgs) == 0 {
		return nil
	}

	qb := sq.Insert("message_outbox").Columns("id", "topic", "key", "payload", "scheduled_at")
	for _, msg := range msgs {
		qb = qb.Values(msg.ID, msg.Topic, msg.Key, msg.Payload, scheduledAt)
	}
	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert messages: %w", err)
	}

	return nil
}

func (s messageOutboxStorage) Delete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sq.
		Delete("message_outbox").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}

	return nil
}

func MessageOutboxMigrations() ([]Migration, error) {
	return []Migration{
		{
			ID: "0000-00-00-001-create-message-outbox-table",
			SQL: `
				create table if not exists message_outbox (
					id           uuid primary key,
					topic        text        not null,
					key          text        not null,
					payload      bytea       not null,
					scheduled_at timestamptz not null
				);

				create index if not exists message_outbox_scheduled_at on message_outbox(scheduled_at)
			`,
		},
	}, nil
}

type sqlxMessage struct {
	ID      uuid.UUID `db:"id"`
	Topic   string    `db:"topic"`
	Key     string    `db:"key"`
	Payload []byte    `db:"payload"`
}
