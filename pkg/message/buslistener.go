package message

import (
	"fmt"

	"github.com/iancoleman/strcase"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type RegisterHandlerFunc func(
	subscriberDomain string,
	deserializer Deserializer,
) (consumerTopic string, consumptionType ConsumptionType, handler Handler, err error)

func RegisterMessageHandler(topic string, consumptionType ConsumptionType, handler Handler) RegisterHandlerFunc {
	return func(_ string, _ Deserializer) (string, ConsumptionType, Handler, error) {
		return topic, consumptionType, handler, nil
	}
}

type (
	HandlerRegistry interface {
		RegisterHandlers(subscriberDomain string, handler RegisterHandlerFunc, handlers ...RegisterHandlerFunc) error
	}

	BusListener interface {
		HandlerRegistry
		Workers() []worker.Process
	}
)

type busListener struct { // TODO: add option to use listener with outbox
	consumers     Consumers
	middlewares   []HandlerMiddleware
	deserializer  Deserializer
	consumersData map[string]consumerData
}

func NewBusListener(
	consumers Consumers,
	handlerMiddlewares ...HandlerMiddleware,
) BusListener {
	return &busListener{
		middlewares:   handlerMiddlewares,
		consumers:     consumers,
		deserializer:  newJSONDeserializer(),
		consumersData: make(map[string]consumerData),
	}
}

func (b *busListener) RegisterHandlers(subscriberDomain string, handler RegisterHandlerFunc, handlers ...RegisterHandlerFunc) error {
	handlers = append([]RegisterHandlerFunc{handler}, handlers...)
	for _, registerFunc := range handlers {
		var err error
		b.consumersData, err = b.registerHandlerFuncImpl(
			subscriberDomain,
			registerFunc,
			b.consumersData,
		)
		if err != nil {
			return fmt.Errorf("register handler func: %w", err)
		}
	}

	return nil
}

func (b *busListener) Workers() []worker.Process {
	workerPool := worker.NewPool(worker.MaxWorkersCountUnlimited)
	listeners := make([]worker.Process, 0, len(b.consumersData))
	for _, data := range b.consumersData {
		listeners = append(listeners,
			NewListener(
				data.Consumer,
				NewCompositeHandler(data.MessageHandlers, workerPool),
				b.middlewares...,
			),
		)
	}

	return listeners
}

func (b *busListener) registerHandlerFuncImpl(
	subscriberDomain string,
	handlerFunc RegisterHandlerFunc,
	consumersData map[string]consumerData,
) (map[string]consumerData, error) {
	consumerTopic, consumptionType, messageHandler, err := handlerFunc(subscriberDomain, b.deserializer)
	if err != nil {
		return nil, fmt.Errorf("execute register func of %v: %w", subscriberDomain, err)
	}

	consumerKey := fmt.Sprintf("%s/%s", subscriberDomain, consumerTopic)
	data, ok := consumersData[consumerKey]
	if ok && data.ConsumptionType != consumptionType {
		return nil, fmt.Errorf(
			"register handler for topic %v and consumption type %v: topic already registered with another consumptionType %v",
			consumerTopic,
			consumptionType,
			data.ConsumptionType,
		)
	}
	if !ok {
		consumer, err := b.consumers.Consumer(consumerTopic, b.getConsumerSubscriptionName(subscriberDomain), consumptionType)
		if err != nil {
			return nil, fmt.Errorf("register consumer for topic %s and consumptionType %s: %w", consumerTopic, consumptionType, err)
		}

		data = consumerData{
			Consumer:        consumer,
			ConsumptionType: consumptionType,
			MessageHandlers: make([]Handler, 0, 1),
		}
	}

	data.MessageHandlers = append(data.MessageHandlers, messageHandler)
	b.consumersData[consumerKey] = data

	return b.consumersData, nil
}

func (b *busListener) getConsumerSubscriptionName(domainName string) string {
	domainName = strcase.ToKebab(domainName)
	return fmt.Sprintf("%s-domain", domainName)
}

type consumerData struct {
	Consumer        Consumer
	ConsumptionType ConsumptionType
	MessageHandlers []Handler
}
