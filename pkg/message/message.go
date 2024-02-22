package message

import (
	"context"

	"github.com/google/uuid"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

/*
TODO перепилить дизайн топиков и сообщений
Сообщения не знают, куда им публиковаться. Вместо этого есть сущность регистри и топика, который создается либо по агрегату+домену, либо по типу сообщения и/или домену, либо как-то еще.
В рамках одного топика уникальны типы сообщений. В реджистри на топики подписываются сообщения, а так же к ним добавляются опции типа количества воркеров (возможно это не реджистри).
Топик знает структуру подписанных сообщений и умеет их сериализовать и наоборот.
Тем самым можно собрать конфигурацию и явно реализовать структуру очередей под паттерны событий (на агрегат), команд (на домен), задач (на тип)
*/

type Message struct {
	ID    uuid.UUID
	Topic string
	// Key is used for topic partitioning, messages with the same key will fall in the same topic partition
	Key     string
	Payload []byte
}

type Handler func(ctx context.Context, msg *Message) error

func NewCompositeHandler(handlers []Handler, optionalPool worker.Pool) Handler {
	if len(handlers) == 0 {
		return func(ctx context.Context, msg *Message) error {
			return nil
		}
	}

	if len(handlers) == 1 {
		handler := handlers[0]
		return func(ctx context.Context, msg *Message) error {
			return handler(ctx, msg)
		}
	}

	if optionalPool == nil {
		return func(ctx context.Context, msg *Message) error {
			for _, handler := range handlers {
				err := handler(ctx, msg)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}

	return func(ctx context.Context, msg *Message) error {
		group := worker.WithinFailSafeGroup(ctx, optionalPool)
		for _, handler := range handlers {
			handlerImpl := handler
			group.Do(func(ctx context.Context) error {
				return handlerImpl(ctx, msg)
			})
		}
		return group.Wait()
	}
}
