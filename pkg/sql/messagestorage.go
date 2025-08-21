package sql

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/message"
)

const messageStorageLockName = "message_storage"

type MessageStorage struct {
	db Client
}

func NewMessageStorage(db Client) *MessageStorage {
	return &MessageStorage{
		db: db,
	}
}

func (s MessageStorage) Lock(ctx context.Context, extraKeys ...string) (context.Context, func() error, error) {
	if len(extraKeys) == 0 {
		return withSessionLevelLock(ctx, messageStorageLockName, s.db)
	}

	sb := strings.Builder{}
	sb.WriteString(messageStorageLockName)
	for _, key := range extraKeys {
		sb.WriteString("_")
		sb.WriteString(key)
	}

	return withSessionLevelLock(ctx, sb.String(), s.db)
}

func (s MessageStorage) Find(ctx context.Context, spec *message.StorageSpecification) ([]message.Message, error) {
	qb := sq.
		Select("id", "topic", "key", "payload").
		From("message_storage").
		Where(sq.LtOrEq{"scheduled_at": spec.ScheduledAtBefore}).
		OrderBy("scheduled_at")
	if len(spec.IDsExcluded) > 0 {
		qb = qb.Where(sq.NotEq{"id": spec.IDsExcluded})
	}
	if len(spec.Topics) > 0 {
		qb = qb.Where(sq.Eq{"topic": spec.Topics})
	}
	if spec.Limit > 0 {
		qb = qb.Limit(uint64(spec.Limit))
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql: %w", err)
	}

	var sqlxResult []sqlxMessage
	err = s.db.SelectContext(ctx, &sqlxResult, query, args...)
	if err != nil {
		return nil, fmt.Errorf("select query: %w", err)
	}

	result := make([]message.Message, 0, len(sqlxResult))
	for _, sqlxMsg := range sqlxResult {
		result = append(result, message.Message{
			ID:      sqlxMsg.ID,
			Topic:   message.Topic(sqlxMsg.Topic),
			Key:     sqlxMsg.Key,
			Payload: sqlxMsg.Payload,
		})
	}

	return result, nil
}

func (s MessageStorage) Store(ctx context.Context, scheduledAt time.Time, msgs ...message.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	qb := sq.Insert("message_storage").Columns("id", "topic", "key", "payload", "scheduled_at")
	for _, msg := range msgs {
		qb = qb.Values(msg.ID, msg.Topic, msg.Key, msg.Payload, scheduledAt)
	}
	qb = qb.Suffix("on conflict (id, topic) do nothing")

	query, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert query: %w", err)
	}

	return nil
}

func (s MessageStorage) Delete(ctx context.Context, topic message.Topic, ids ...uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sq.
		Delete("message_storage").
		Where(sq.Eq{"topic": topic}).
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete query: %w", err)
	}

	return nil
}

func MessageStorageMigrations() ([]Migration, error) {
	return []Migration{
		{
			ID: "0000-00-00-001-create-message-storage-table",
			SQL: `
				create table if not exists message_storage (
					id           uuid        not null,
					topic        text        not null,
					key          text        not null,
					payload      bytea       not null,
					scheduled_at timestamptz not null,
					primary key (id, topic)
				);

				create index if not exists message_storage_scheduled_at_topic on message_storage(scheduled_at, topic);
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
