package message

import (
	"fmt"

	"github.com/klwxsrx/go-service-template/pkg/worker"
)

type (
	TopicHandlersMap map[TopicSubscription]TopicHandlers
	TopicHandlers    []RegisterHandlerFunc

	HandlerRegistry interface {
		RegisterMessageHandlers(SubscriberName, TopicHandlersMap) error
	}

	BusListener interface {
		HandlerRegistry
		Workers() []worker.ErrorJob
	}

	TopicSubscription struct {
		Topic           Topic
		ConsumptionType ConsumptionType
	}

	RegisterHandlerFunc func() (StructuredMessage, DeserializerFunc, TypedHandler[StructuredMessage])
)

type busListener struct {
	consumers     ConsumerProvider
	middlewares   []HandlerMiddleware
	consumersData map[string]consumerData
}

func NewBusListener(
	consumers ConsumerProvider,
	handlerMiddlewares ...HandlerMiddleware,
) BusListener {
	return &busListener{
		middlewares:   handlerMiddlewares,
		consumers:     consumers,
		consumersData: make(map[string]consumerData),
	}
}

func (l *busListener) RegisterMessageHandlers(subscriber SubscriberName, topicHandlers TopicHandlersMap) error {
	for topicSubscription, handlers := range topicHandlers {
		for _, registerFunc := range handlers {
			var err error
			l.consumersData, err = l.registerHandlerFuncImpl(
				subscriber,
				topicSubscription,
				registerFunc,
				l.consumersData,
			)
			if err != nil {
				return fmt.Errorf("register handler func: %w", err)
			}
		}
	}

	return nil
}

func (l *busListener) Workers() []worker.ErrorJob {
	workerPool := worker.NewPool(worker.MaxWorkersCountUnlimited)
	listeners := make([]worker.ErrorJob, 0, len(l.consumersData))
	for _, data := range l.consumersData {
		listeners = append(listeners,
			newListener(
				data.Consumer,
				data.Deserializer,
				NewCompositeHandler(data.MessageHandlers, workerPool),
				l.middlewares...,
			),
		)
	}

	return listeners
}

func (l *busListener) registerHandlerFuncImpl(
	subscriber SubscriberName,
	subscription TopicSubscription,
	handler RegisterHandlerFunc,
	consumersData map[string]consumerData,
) (map[string]consumerData, error) {
	consumerKey := fmt.Sprintf("%s/%s", subscriber, subscription.Topic)
	data, ok := consumersData[consumerKey]
	if ok && data.ConsumptionType != subscription.ConsumptionType {
		return nil, fmt.Errorf(
			"register handler for topic %v and consumption type %v: topic already registered with another consumptionType %v",
			subscription.Topic,
			subscription.ConsumptionType,
			data.ConsumptionType,
		)
	}

	if !ok {
		consumer, err := l.consumers.Consumer(subscription.Topic, subscriber, subscription.ConsumptionType)
		if err != nil {
			return nil, fmt.Errorf(
				"register consumer for topic %s: provide consumer: %w",
				subscription.Topic,
				err,
			)
		}

		data = consumerData{
			Consumer:        consumer,
			ConsumptionType: subscription.ConsumptionType,
			Deserializer:    newJSONDeserializer(),
			MessageHandlers: make([]TypedHandler[StructuredMessage], 0, 1),
		}
	}

	messageSchema, deserializerFunc, messageHandler := handler()
	messageType := messageSchema.Type()
	if messageType == "" {
		return nil, fmt.Errorf("get message type for %T: blank message must return const value", messageSchema)
	}

	err := data.Deserializer.Register(messageType, deserializerFunc)
	if err != nil {
		return nil, fmt.Errorf(
			"register consumer for topic %s: register deserializer: %w",
			subscription.Topic,
			err,
		)
	}

	data.MessageHandlers = append(data.MessageHandlers, messageHandler)
	l.consumersData[consumerKey] = data

	return l.consumersData, nil
}

type consumerData struct {
	Consumer        Consumer
	ConsumptionType ConsumptionType
	Deserializer    jsonDeserializer
	MessageHandlers []TypedHandler[StructuredMessage]
}
