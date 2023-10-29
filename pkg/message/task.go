package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/task"
)

const messageClassTask = "task"

type taskScheduler struct {
	bus Bus
}

func NewTaskScheduler(bus Bus) task.Scheduler {
	return taskScheduler{
		bus: bus,
	}
}

func (s taskScheduler) Schedule(ctx context.Context, at time.Time, tasks ...task.Task) error {
	for _, tsk := range tasks {
		err := s.bus.Produce(ctx, messageClassTask, tsk, at)
		if err != nil {
			return fmt.Errorf("publish task: %w", err)
		}
	}
	return nil
}

func RegisterTask[T task.Task]() RegisterStructuredMessageFunc {
	return func(domainName string) (messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc, err error) {
		var blank T
		taskType := blank.Type()
		if taskType == "" {
			return "",
				"",
				nil,
				nil,
				fmt.Errorf("get task type for %T: blank task must return const value", blank)
		}

		return messageClassTask,
			taskType,
			func(string) string {
				return buildTaskTopic(domainName, taskType)
			},
			func(StructuredMessage) string {
				return ""
			},
			nil
	}
}

func RegisterTaskHandler[T task.Task](handler task.TypedHandler[T]) RegisterHandlerFunc {
	return func(subscriberDomain string, deserializer Deserializer) (string, ConsumptionType, Handler, error) {
		var blank T
		taskType := blank.Type()
		if taskType == "" {
			return "",
				"",
				nil,
				fmt.Errorf("get task type for %T: blank task must return const value", blank)
		}

		err := deserializer.RegisterDeserializer(subscriberDomain, messageClassTask, taskType, TypedDeserializer[T]())
		if err != nil {
			return "",
				"",
				nil,
				fmt.Errorf("register task %T deserializer: %w", blank, err)
		}

		return buildTaskTopic(subscriberDomain, taskType),
			ConsumptionTypeShared,
			taskHandlerImpl[T](subscriberDomain, handler, deserializer),
			nil
	}
}

func buildTaskTopic(domainName, taskType string) string {
	domainName = strcase.ToKebab(domainName)
	taskType = strcase.ToKebab(taskType)
	return fmt.Sprintf("task.%s-domain.%s-queue", domainName, taskType)
}

func taskHandlerImpl[T task.Task](
	publisherDomain string,
	handler task.TypedHandler[T],
	deserializer Deserializer,
) Handler {
	return func(ctx context.Context, msg *Message) error {
		tsk, err := deserializer.Deserialize(ctx, publisherDomain, messageClassTask, msg)
		if errors.Is(err, ErrDeserializeNotValidMessage) || errors.Is(err, ErrDeserializeUnknownMessage) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("deserialize message %v: %w", msg.ID, err)
		}

		concreteTask, ok := tsk.(T)
		if !ok {
			return fmt.Errorf("invalid task struct type %T for messageID %v, expected %T", tsk, msg.ID, concreteTask)
		}

		return handler(ctx, concreteTask)
	}
}
