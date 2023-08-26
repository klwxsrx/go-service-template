package sql

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/message"
)

const (
	batchLimit = 500

	messageStoreTableDDL = `
		CREATE TABLE IF NOT EXISTS message_outbox (
			id      uuid PRIMARY KEY,
			topic   text,
			key     text,
			payload bytea,
			created_at timestamptz default current_timestamp
		)
	`
	messageStoreTableIndexDDL = `
		CREATE INDEX IF NOT EXISTS message_outbox_created_at ON message_outbox(created_at)
	`
)

type messageStore struct {
	db Client
}

func (s *messageStore) GetBatch(ctx context.Context) ([]message.Message, error) {
	query, args, err := sq.
		Select("id", "topic", "key", "payload").
		From("message_outbox").
		OrderBy("created_at").
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

func (s *messageStore) Store(ctx context.Context, msgs []message.Message) error {
	qb := sq.Insert("message_outbox").Columns("id", "topic", "key", "payload")
	for _, msg := range msgs {
		qb = qb.Values(msg.ID, msg.Topic, msg.Key, msg.Payload)
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

func (s *messageStore) Delete(ctx context.Context, ids []uuid.UUID) error {
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

func (s *messageStore) createMessageStoreTableIfNotExists(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, messageStoreTableDDL)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, messageStoreTableIndexDDL)
	return err
}

func NewMessageStore(ctx context.Context, db Client) (message.Store, error) {
	s := &messageStore{db: db}
	err := s.createMessageStoreTableIfNotExists(ctx)
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
