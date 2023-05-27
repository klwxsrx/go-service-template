package message

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/klwxsrx/go-service-template/pkg/event"
	"github.com/klwxsrx/go-service-template/pkg/hub"
	"strings"
	"unicode"
)

type (
	RegisterHandlerFunc func(domainName string, deserializer *jsonEventDeserializer) (consumerTopic string, consumptionType ConsumptionType, handler Handler, err error)

	HandlerOption func(eventDeserializer *jsonEventDeserializer) HandlerMiddleware
)

func RegisterEventHandler[T event.Event](handler event.TypedHandler[T]) RegisterHandlerFunc {
	return func(domainName string, eventDeserializer *jsonEventDeserializer) (string, ConsumptionType, Handler, error) {
		var blankEvent T
		aggregateName := blankEvent.AggregateName()
		if aggregateName == "" {
			return "", "", nil, fmt.Errorf("failed to get aggregate name for %T: blank event must return const value", blankEvent)
		}

		err := eventDeserializer.RegisterJSONEvent(domainName, registerDeserializerTyped[T]())
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to register event deserializer: %w", err)
		}

		return getEventTopic(domainName, aggregateName),
			ConsumptionTypeSingle,
			eventHandlerImpl[T](domainName, handler, eventDeserializer),
			nil
	}
}

// TODO: RegisterCommandHandler

func RegisterMessageHandler(topic string, consumptionType ConsumptionType, handler Handler) RegisterHandlerFunc {
	return func(_ string, _ *jsonEventDeserializer) (string, ConsumptionType, Handler, error) {
		return topic, consumptionType, handler, nil
	}
}

func Must[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}

type HandlerRegistry interface {
	Register(domainName, publisherDomainName string, handler RegisterHandlerFunc, handlers ...RegisterHandlerFunc)
}

type ListenerManager interface {
	HandlerRegistry
	Listeners() ([]hub.Process, error)
}

type listenerManager struct {
	panicHandler     PanicHandler
	middlewares      []HandlerMiddleware
	consumerProvider ConsumerProvider
	handlerRegisters map[handlerDomainData][]RegisterHandlerFunc
}

func (m *listenerManager) Register(domainName, publisherDomainName string, handlerFunc RegisterHandlerFunc, handlerFuncs ...RegisterHandlerFunc) {
	handlerFuncs = append([]RegisterHandlerFunc{handlerFunc}, handlerFuncs...)
	domainData := handlerDomainData{
		DomainName:          domainName,
		PublisherDomainName: publisherDomainName,
	}
	m.handlerRegisters[domainData] = append(m.handlerRegisters[domainData], handlerFuncs...)
}

func (m *listenerManager) Listeners() ([]hub.Process, error) {
	eventDeserializer := newJSONEventDeserializer()
	consumers := make(map[string]consumerData)
	for domainData, registerFuncs := range m.handlerRegisters {
		for _, registerFunc := range registerFuncs {
			var err error
			consumers, err = m.registerHandlerFuncImpl(
				domainData.DomainName,
				domainData.PublisherDomainName,
				registerFunc,
				eventDeserializer,
				consumers,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	listeners := make([]hub.Process, 0, len(consumers))
	for _, data := range consumers {
		listeners = append(listeners,
			NewListener(
				NewCompositeHandler(nil, data.MessageHandlers),
				data.Consumer,
				m.panicHandler,
				m.middlewares...,
			),
		)
	}

	return listeners, nil
}

func (m *listenerManager) registerHandlerFuncImpl(
	domainName string,
	publisherDomainName string,
	handlerFunc RegisterHandlerFunc,
	eventDeserializer *jsonEventDeserializer,
	consumers map[string]consumerData,
) (map[string]consumerData, error) {
	consumerTopic, consumptionType, messageHandler, err := handlerFunc(publisherDomainName, eventDeserializer)
	if err != nil {
		return nil, fmt.Errorf("failed to execute register func of %v to publisher %v: %w", domainName, publisherDomainName, err)
	}

	consumerKey := fmt.Sprintf("%s/%s", domainName, consumerTopic)
	data, ok := consumers[consumerKey]
	if ok && data.ConsumptionType != consumptionType {
		return nil, fmt.Errorf(
			"failed to register handler for topic %v and consumption type %v, topic already registered with another consumptionType %v",
			consumerTopic,
			consumptionType,
			data.ConsumptionType,
		)
	}
	if !ok {
		consumer, err := m.consumerProvider.ProvideConsumer(consumerTopic, m.getConsumerSubscriptionName(domainName), consumptionType)
		if err != nil {
			return nil, fmt.Errorf("failed to register consumer for topic %s and consumptionType %s: %w", consumerTopic, consumptionType, err)
		}

		data = consumerData{
			Consumer:        consumer,
			ConsumptionType: consumptionType,
			MessageHandlers: make([]Handler, 0, 1),
		}
	}

	data.MessageHandlers = append(data.MessageHandlers, messageHandler)
	consumers[consumerKey] = data

	return consumers, nil
}

func (m *listenerManager) getConsumerSubscriptionName(domainName string) string {
	domainName = prepareNameToKebabCase(domainName)
	return fmt.Sprintf("%s-domain", domainName)
}

func NewListenerManager(
	consumerProvider ConsumerProvider,
	panicHandler PanicHandler,
	handlerMiddlewares ...HandlerMiddleware,
) ListenerManager {
	return &listenerManager{
		panicHandler:     panicHandler,
		middlewares:      handlerMiddlewares,
		consumerProvider: consumerProvider,
		handlerRegisters: make(map[handlerDomainData][]RegisterHandlerFunc),
	}
}

func getEventTopic(domainName, eventAggregateName string) string {
	domainName = prepareNameToKebabCase(domainName)
	aggregateName := prepareNameToKebabCase(eventAggregateName)
	return fmt.Sprintf("event.%s-domain.%s-aggregate", domainName, aggregateName)
}

func prepareNameToKebabCase(name string) string {
	return strcase.ToKebab(
		strings.Map(func(r rune) rune {
			if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) || r == '_' || r == '-' {
				return r
			}
			return -1
		}, name),
	)
}

type handlerDomainData struct {
	DomainName          string
	PublisherDomainName string
}

type consumerData struct {
	Consumer        Consumer
	ConsumptionType ConsumptionType
	MessageHandlers []Handler
}
