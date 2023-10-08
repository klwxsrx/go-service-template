package message

import (
	"fmt"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type RegisterHandlerFunc func(
	publisherDomain string,
	deserializer Deserializer,
) (consumerTopic string, consumptionType ConsumptionType, handler Handler, err error)

func RegisterMessageHandler(topic string, consumptionType ConsumptionType, handler Handler) RegisterHandlerFunc {
	return func(_ string, _ Deserializer) (string, ConsumptionType, Handler, error) {
		return topic, consumptionType, handler, nil
	}
}

func Must[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}

type (
	HandlerRegistry interface {
		RegisterHandler(subscriberDomain, publisherDomain string, handler RegisterHandlerFunc, handlers ...RegisterHandlerFunc)
	}

	BusListener interface {
		HandlerRegistry
		ListenerWorkers() ([]worker.NamedProcess, error)
	}
)

type busListener struct {
	middlewares      []HandlerMiddleware
	consumers        Consumers
	handlerRegisters map[handlerData][]RegisterHandlerFunc
}

func NewBusListener(
	consumers Consumers,
	handlerMiddlewares ...HandlerMiddleware, // TODO: add observability to pass request id
) BusListener {
	return &busListener{
		middlewares:      handlerMiddlewares,
		consumers:        consumers,
		handlerRegisters: make(map[handlerData][]RegisterHandlerFunc),
	}
}

func (b *busListener) RegisterHandler(subscriberDomain, publisherDomain string, handler RegisterHandlerFunc, handlers ...RegisterHandlerFunc) {
	handlers = append([]RegisterHandlerFunc{handler}, handlers...)

	handlerID := handlerData{
		SubscriberDomain: subscriberDomain,
		PublisherDomain:  publisherDomain,
	}
	b.handlerRegisters[handlerID] = append(b.handlerRegisters[handlerID], handlers...)
}

func (b *busListener) ListenerWorkers() ([]worker.NamedProcess, error) {
	deserializer := newJSONDeserializer()
	consumers := make(map[string]consumerData)
	for domainData, registerFuncs := range b.handlerRegisters {
		for _, registerFunc := range registerFuncs {
			var err error
			consumers, err = b.registerHandlerFuncImpl(
				domainData.SubscriberDomain,
				domainData.PublisherDomain,
				registerFunc,
				consumers,
				deserializer,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	listeners := make([]worker.NamedProcess, 0, len(consumers))
	for _, data := range consumers {
		listeners = append(listeners,
			NewListener(
				data.Consumer,
				NewCompositeHandler(data.MessageHandlers, nil),
				b.middlewares...,
			),
		)
	}

	return listeners, nil
}

func (b *busListener) registerHandlerFuncImpl(
	subscriberDomain string,
	publisherDomain string,
	handlerFunc RegisterHandlerFunc,
	consumers map[string]consumerData,
	deserializer Deserializer,
) (map[string]consumerData, error) {
	consumerTopic, consumptionType, messageHandler, err := handlerFunc(publisherDomain, deserializer)
	if err != nil {
		return nil, fmt.Errorf("failed to execute register func of %v to publisher %v: %w", subscriberDomain, publisherDomain, err)
	}

	consumerKey := fmt.Sprintf("%s/%s", subscriberDomain, consumerTopic)
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
		consumer, err := b.consumers.Consumer(consumerTopic, getConsumerSubscriptionName(subscriberDomain), consumptionType)
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

func getConsumerSubscriptionName(domainName string) string {
	domainName = strcase.ToKebab(domainName)
	return fmt.Sprintf("%s-domain", domainName)
}

type (
	handlerData struct {
		SubscriberDomain string
		PublisherDomain  string
	}

	consumerData struct {
		Consumer        Consumer
		ConsumptionType ConsumptionType
		MessageHandlers []Handler
	}
)
