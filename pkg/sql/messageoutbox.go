package sql

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/message"
)

const (
	batchLimit = 500

	messageOutboxStorageTableDDL = `
		CREATE TABLE IF NOT EXISTS message_outbox (
			id           uuid PRIMARY KEY,
			topic        text        NOT NULL,
			key          text        NOT NULL,
			payload      bytea       NOT NULL,
			scheduled_at timestamptz NOT NULL
		)
	`
	messageOutboxStorageTableIndexDDL = `
		CREATE INDEX IF NOT EXISTS message_outbox_scheduled_at ON message_outbox(scheduled_at)
	`
)

type messageOutboxStorage struct {
	db Client
}

func (s *messageOutboxStorage) GetBatch(ctx context.Context, scheduledBefore time.Time) ([]message.Message, error) {
	query, args, err := sq.
		Select("id", "topic", "key", "payload").
		From("message_outbox").
		Where(sq.LtOrEq{"scheduled_at": scheduledBefore}).
		OrderBy("scheduled_at").
		Limit(batchLimit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build sql: %w", err)
	}

	var sqlxResult []sqlxMessage
	err = s.db.SelectContext(ctx, &sqlxResult, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select messages: %w", err)
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

func (s *messageOutboxStorage) Store(ctx context.Context, msgs []message.Message, scheduledAt time.Time) error {
	qb := sq.Insert("message_outbox").Columns("id", "topic", "key", "payload", "scheduled_at")
	for _, msg := range msgs {
		qb = qb.Values(msg.ID, msg.Topic, msg.Key, msg.Payload, scheduledAt)
	}
	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert messages: %w", err)
	}

	return nil
}

func (s *messageOutboxStorage) Delete(ctx context.Context, ids []uuid.UUID) error {
	query, args, err := sq.
		Delete("message_outbox").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}

func (s *messageOutboxStorage) createStorageTableIfNotExists(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, messageOutboxStorageTableDDL)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, messageOutboxStorageTableIndexDDL)
	return err
}

func NewMessageOutboxStorage(ctx context.Context, db Client) (message.OutboxStorage, error) {
	s := &messageOutboxStorage{db: db}
	err := s.createStorageTableIfNotExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create message store table: %w", err)
	}
	return s, nil
}

type sqlxMessage struct {
	ID      uuid.UUID `db:"id"`
	Topic   string    `db:"topic"`
	Key     string    `db:"key"`
	Payload []byte    `db:"payload"`
}
