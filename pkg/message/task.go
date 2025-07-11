package message

import (
	"context"
	"fmt"
	"time"

	"github.com/klwxsrx/go-service-template/pkg/task"
)

type (
	TaskScheduler interface {
		task.Scheduler
		ProducerRegistry
	}

	taskScheduler struct {
		bus BusProducer
	}
)

func NewTaskScheduler(
	bus BusProducer,
) TaskScheduler {
	return taskScheduler{
		bus: bus,
	}
}

func (s taskScheduler) Schedule(ctx context.Context, at time.Time, tasks ...task.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	msgs := make([]StructuredMessage, 0, len(tasks))
	for _, tsk := range tasks {
		msgs = append(msgs, StructuredMessage(tsk))
	}

	err := s.bus.Produce(ctx, msgs, at)
	if err != nil {
		return fmt.Errorf("publish task: %w", err)
	}

	return nil
}

func (s taskScheduler) RegisterMessages(messagesMap TopicMessagesMap) error {
	return s.bus.RegisterMessages(messagesMap)
}

func RegisterTask[T task.Task]() RegisterMessageFunc {
	return func() (StructuredMessage, KeyBuilderFunc) {
		var blank T
		return blank, nil
	}
}

func RegisterTaskHandler[T task.Task](handler task.TypedHandler[T]) RegisterHandlerFunc {
	return func() (StructuredMessage, Deserializer, TypedHandler[StructuredMessage]) {
		var blank T
		return blank, TypedJSONDeserializer[T](), func(ctx context.Context, msg StructuredMessage) error {
			tsk, ok := msg.(T)
			if !ok {
				return fmt.Errorf("invalid task struct type %T for messageID %v, expected %T", msg, msg.ID(), tsk)
			}

			return handler(ctx, tsk)
		}
	}
}

func NewTopicTaskQueue(domainName, taskType string, customTags ...string) Topic {
	return NewTopic(
		"task-queue",
		WithTopicDomainName(domainName),
		WithTopicMessageType(taskType),
		WithTopicCustomTags(customTags...),
	)
}

func NewTopicSubscriptionTaskQueue(domainName, taskType string, customTags ...string) TopicSubscription {
	return TopicSubscription{
		Topic:           NewTopicTaskQueue(domainName, taskType, customTags...),
		ConsumptionType: ConsumptionTypeShared,
	}
}
