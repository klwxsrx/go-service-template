package pulsar

import (
	"fmt"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/cenkalti/backoff/v4"

	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
)

const defaultConnectionTimeout = 20 * time.Second

type Config struct {
	Address           string
	ConnectionTimeout time.Duration
}

type MessageBroker struct { // TODO: change to kafka
	client pulsar.Client

	producerMutexes *sync.Map
	producers       map[message.Topic]pulsar.Producer
}

func NewMessageBroker(config *Config, logger log.Logger) (*MessageBroker, error) {
	c, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:    fmt.Sprintf("pulsar://%s", config.Address),
		Logger: newLoggerAdapter(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("create pulsar client: %w", err)
	}

	conn := &MessageBroker{
		client:          c,
		producerMutexes: &sync.Map{},
		producers:       make(map[message.Topic]pulsar.Producer),
	}

	connTimeout := defaultConnectionTimeout
	if config.ConnectionTimeout > 0 {
		connTimeout = config.ConnectionTimeout
	}
	err = conn.testCreateProducer(connTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to broker: %w", err)
	}

	return conn, nil
}

func (b *MessageBroker) Close() {
	for _, producer := range b.producers {
		producer.Close()
	}
	b.client.Close()
}

func (b *MessageBroker) testCreateProducer(connTimeout time.Duration) error {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = time.Second
	eb.RandomizationFactor = 0
	eb.Multiplier = 2
	eb.MaxInterval = connTimeout / 4
	eb.MaxElapsedTime = connTimeout

	return backoff.Retry(func() error {
		p, err := b.client.CreateProducer(pulsar.ProducerOptions{
			Topic: "non-persistent://public/default/test-topic",
		})
		if err == nil {
			p.Close()
		}
		return err
	}, eb)
}
