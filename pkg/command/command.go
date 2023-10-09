//go:generate ${TOOLS_PATH}/mockgen -source ${GOFILE} -destination mock/${GOFILE} -package mock -mock_names "Command=Command,Bus=Bus"
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Command interface {
	ID() uuid.UUID
	Type() string
}

type (
	TypedHandler[T Command] func(ctx context.Context, command T) error
	Handler                 TypedHandler[Command]

	RegisterHandlerFunc func() (commandType string, handler Handler, err error)
)

type Bus interface {
	Publish(ctx context.Context, commands []Command) error
}

type bus struct {
	handlers map[string]Handler
}

func (b bus) Publish(ctx context.Context, commands []Command) error {
	for _, command := range commands {
		handler, ok := b.handlers[command.Type()]
		if !ok {
			return fmt.Errorf("handler not registered for %s", command.Type())
		}

		err := handler(ctx, command)
		if err != nil {
			return fmt.Errorf("failed to execute command: %w", err)
		}
	}
	return nil
}

func RegisterHandler[T Command](handler TypedHandler[T]) RegisterHandlerFunc {
	return func() (string, Handler, error) {
		var blankCommand T
		commandType := blankCommand.Type()
		if commandType == "" {
			return "", nil, fmt.Errorf("failed to get command type for %T: blank command must return const value", blankCommand)
		}

		return commandType, func(ctx context.Context, command Command) error {
			concreteCommand, ok := command.(T)
			if !ok {
				return fmt.Errorf("invalid command struct type %T, expected %T", command, concreteCommand)
			}
			return handler(ctx, concreteCommand)
		}, nil
	}
}

func NewBus(handlers ...RegisterHandlerFunc) (Bus, error) {
	handlersMap := make(map[string]Handler, len(handlers))
	for _, registerFunc := range handlers {
		commandType, handler, err := registerFunc()
		if err != nil {
			return nil, err
		}
		if _, ok := handlersMap[commandType]; ok {
			return nil, fmt.Errorf("command handler for %s already exists", commandType)
		}

		handlersMap[commandType] = handler
	}

	return bus{handlers: handlersMap}, nil
}
