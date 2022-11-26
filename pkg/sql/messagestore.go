package sql

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
)

const (
	batchLimit = 1000
)

type messageStore struct {
	db Client
}

func (s *messageStore) GetBatch(ctx context.Context) ([]message.Message, error) {
	query, args, err := sq.
		Select("id", "topic", "key", "payload").
		From("message_outbox").
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

	result := make([]message.Message, len(sqlxResult))
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

func NewMessageStore(db Client) persistence.MessageStore {
	return &messageStore{db: db}
}

type sqlxMessage struct {
	ID      uuid.UUID `db:"id"`
	Topic   string    `db:"topic"`
	Key     string    `db:"key"`
	Payload []byte    `db:"payload"`
}
