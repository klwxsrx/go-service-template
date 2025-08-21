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
		Registry
	}

	taskScheduler struct {
		bus BusScheduledProducer
	}
)

func NewTaskScheduler(
	bus BusScheduledProducer,
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

	err := s.bus.Schedule(ctx, at, msgs...)
	if err != nil {
		return fmt.Errorf("publish task: %w", err)
	}

	return nil
}

func (s taskScheduler) Register(messages TopicMessages, opts ...BusProducerOption) error {
	return s.bus.Register(messages, opts...)
}

func RegisterTask[T task.Task]() RegisterMessageFunc {
	return func() (StructuredMessage, KeyBuilder) {
		var blank T
		return blank, nil
	}
}

func RegisterTaskHandlers[T task.Task](handlers ...task.TypedHandler[T]) RegisterHandlersFunc {
	return func() (StructuredMessage, PayloadDeserializer, []TypedHandler[StructuredMessage]) {
		handlersImpl := make([]TypedHandler[StructuredMessage], 0, len(handlers))
		for _, handler := range handlers {
			handlersImpl = append(handlersImpl, func(ctx context.Context, msg StructuredMessage) error {
				tsk, ok := msg.(T)
				if !ok {
					return fmt.Errorf("invalid task struct type %T for messageID %v, expected %T", msg, msg.ID(), tsk)
				}

				return handler(ctx, tsk)
			})
		}

		var blank T
		return blank, PayloadDeserializerImpl[T], handlersImpl
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
