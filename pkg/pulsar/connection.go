package pulsar

import (
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/cenkalti/backoff/v4"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"time"
)

const defaultConnectionTimeout = 20 * time.Second

type ConsumptionType int

const (
	ConsumptionTypeFailover ConsumptionType = iota
	ConsumptionTypeShared
)

type Config struct {
	Address           string
	AuthToken         string
	ConnectionTimeout time.Duration
}

type ConsumerOptions struct {
	Topic            string
	SubscriptionName string
	ConsumptionType  ConsumptionType
}

type Connection interface {
	Producer(topic string) (message.Producer, error)
	Consumer(config *ConsumerOptions) (message.Consumer, error)
	Close()
}

type connection struct {
	client pulsar.Client
}

func (c *connection) Producer(topic string) (message.Producer, error) {
	p, err := c.client.CreateProducer(pulsar.ProducerOptions{
		Topic: topic,
	})
	if err != nil {
		return nil, err
	}
	return newMessageProducer(p), nil
}

func (c *connection) Consumer(config *ConsumerOptions) (message.Consumer, error) {
	typeOpt := pulsar.Failover
	switch config.ConsumptionType {
	case ConsumptionTypeFailover:
		typeOpt = pulsar.Failover
	case ConsumptionTypeShared:
		typeOpt = pulsar.Shared
	default:
	}

	opts := pulsar.ConsumerOptions{
		Topic:            config.Topic,
		SubscriptionName: config.SubscriptionName,
		Type:             typeOpt,
	}
	cons, err := c.client.Subscribe(opts)
	if err != nil {
		return nil, err
	}
	return newMessageConsumer(cons), nil
}

func (c *connection) Close() {
	c.client.Close()
}

func (c *connection) testCreateProducer(connTimeout time.Duration) error {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = time.Second
	eb.RandomizationFactor = 0
	eb.Multiplier = 2
	eb.MaxInterval = connTimeout / 4
	eb.MaxElapsedTime = connTimeout

	return backoff.Retry(func() error {
		p, err := c.client.CreateProducer(pulsar.ProducerOptions{
			Topic: "non-persistent://public/default/test-topic",
		})
		if err == nil {
			p.Close()
		}
		return err
	}, eb)
}

func NewConnection(config *Config, logger log.Logger) (Connection, error) {
	c, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            fmt.Sprintf("pulsar://%s", config.Address),
		Authentication: pulsar.NewAuthenticationToken(config.AuthToken),
		Logger:         newLoggerAdapter(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	conn := &connection{client: c}
	connTimeout := defaultConnectionTimeout
	if config.ConnectionTimeout > 0 {
		connTimeout = config.ConnectionTimeout
	}

	err = conn.testCreateProducer(connTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to broker: %w", err)
	}

	return conn, nil
}
