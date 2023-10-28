package message

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/command"
)

const messageClassCommand = "command"

type commandBus struct {
	bus Bus
}

func NewCommandBus(bus Bus) command.Bus {
	return commandBus{
		bus: bus,
	}
}

func (b commandBus) Publish(ctx context.Context, commands ...command.Command) error {
	for _, cmd := range commands {
		err := b.bus.Produce(ctx, messageClassCommand, cmd, time.Now())
		if err != nil {
			return fmt.Errorf("failed to publish command: %w", err)
		}
	}
	return nil
}

func RegisterCommand[T command.Command]() RegisterStructuredMessageFunc {
	return func(domainName string) (messageClass, messageType string, topicBuilder TopicBuilderFunc, keyBuilder KeyBuilderFunc, err error) {
		var blank T
		commandType := blank.Type()
		if commandType == "" {
			return "",
				"",
				nil,
				nil,
				fmt.Errorf("failed to get command type for %T: blank command must return const value", blank)
		}

		return messageClassCommand,
			commandType,
			buildCommandTopic,
			func(StructuredMessage) string {
				return ""
			},
			nil
	}
}

func RegisterCommandHandler[T command.Command](handler command.TypedHandler[T]) RegisterHandlerFunc {
	return func(subscriberDomain string, deserializer Deserializer) (string, ConsumptionType, Handler, error) {
		var blank T
		commandType := blank.Type()
		if commandType == "" {
			return "",
				"",
				nil,
				fmt.Errorf("failed to get command type for %T: blank command must return const value", blank)
		}

		err := deserializer.RegisterDeserializer(subscriberDomain, messageClassCommand, commandType, TypedDeserializer[T]())
		if err != nil {
			return "",
				"",
				nil,
				fmt.Errorf("failed to register command %T deserializer: %w", blank, err)
		}

		return buildCommandTopic(subscriberDomain),
			ConsumptionTypeSingle,
			commandHandlerImpl[T](subscriberDomain, handler, deserializer),
			nil
	}
}

func buildCommandTopic(domainName string) string {
	domainName = strcase.ToKebab(domainName)
	return fmt.Sprintf("command.%s-domain", domainName)
}

func commandHandlerImpl[T command.Command](
	publisherDomain string,
	handler command.TypedHandler[T],
	deserializer Deserializer,
) Handler {
	return func(ctx context.Context, msg *Message) error {
		cmd, err := deserializer.Deserialize(ctx, publisherDomain, messageClassCommand, msg)
		if errors.Is(err, ErrDeserializeNotValidMessage) || errors.Is(err, ErrDeserializeUnknownMessage) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to deserialize message %v: %w", msg.ID, err)
		}

		concreteCommand, ok := cmd.(T)
		if !ok {
			return fmt.Errorf("invalid command struct type %T for messageID %v, expected %T", cmd, msg.ID, concreteCommand)
		}

		return handler(ctx, concreteCommand)
	}
}
