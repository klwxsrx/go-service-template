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

const defaultConnectionTimeout = 10 * time.Second

type Config struct {
	Address           string
	ConnectionTimeout time.Duration
}

type MessageBroker struct {
	client pulsar.Client

	producerMutexes *sync.Map
	producers       map[message.Topic]pulsar.Producer
}

func NewMessageBroker(config *Config) (*MessageBroker, error) {
	c, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:    fmt.Sprintf("pulsar://%s", config.Address),
		Logger: newLoggerAdapter(log.New(log.LevelDisabled)),
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

func (b *MessageBroker) Close() error {
	for _, producer := range b.producers {
		producer.Close()
	}

	b.client.Close()
	return nil
}

func (b *MessageBroker) testCreateProducer(connTimeout time.Duration) error {
	eb := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(time.Second),
		backoff.WithMultiplier(2),
		backoff.WithMaxInterval(connTimeout/4),
		backoff.WithMaxElapsedTime(connTimeout),
	)

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
