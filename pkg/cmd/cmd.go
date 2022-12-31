package cmd

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/log"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
	"io/fs"
)

type App interface {
	Finish(context.Context)
}

type app struct {
	logger log.Logger
}

func (a *app) Finish(ctx context.Context) {
	msg := recover()
	if msg == nil {
		return
	}

	logger := a.logger
	err, ok := msg.(error)
	if ok {
		logger = logger.WithError(err)
	} else {
		logger = logger.WithField("err", msg)
	}
	logger.Fatal(ctx, "app failed with panic")
}

func StartApp(logLevel log.Level) (App, context.Context, log.Logger) {
	ctx := context.Background()
	logger := log.New(logLevel)
	return &app{logger}, ctx, logger
}

func MustInitSQL(ctx context.Context, logger log.Logger, optionalMigrations fs.ReadDirFS) sql.Connection {
	sqlConfig := &sql.Config{
		DSN: sql.DSN{
			User:     env.Must(env.ParseString("SQL_USER")),
			Password: env.Must(env.ParseString("SQL_PASSWORD")),
			Address:  env.Must(env.ParseString("SQL_ADDRESS")),
			Database: env.Must(env.ParseString("SQL_DATABASE")),
		},
	}
	sqlConnTimeout, err := env.ParseDuration("SQL_CONNECTION_TIMEOUT")
	if err == nil {
		sqlConfig.ConnectionTimeout = sqlConnTimeout
	}

	sqlConn, err := sql.NewConnection(sqlConfig, logger)
	if err != nil {
		panicInitApplication(err)
	}

	if optionalMigrations == nil {
		return sqlConn
	}
	sqlMigration := sql.NewMigration(sqlConn.Client(), optionalMigrations, logger)
	err = sqlMigration.Execute(ctx)
	if err != nil {
		panicInitApplication(err)
	}
	return sqlConn
}

func MustInitSQLTransaction(
	sqlConn sql.Connection,
	onCommit func(),
) (sql.Client, persistence.Transaction) {
	return sql.NewTransaction(sqlConn.Client(), onCommit)
}

func MustInitSQLMessageOutbox(
	ctx context.Context,
	sqlConn sql.Connection,
	producers pkgmessage.ProducerProvider,
	logger log.Logger,
) pkgmessage.Outbox {
	sqlClient, tx := MustInitSQLTransaction(sqlConn, func() {})
	messageStore, err := sql.NewMessageStore(ctx, sqlClient)
	if err != nil {
		panicInitApplication(err)
	}
	return pkgmessage.NewOutbox(
		pkgmessage.NewSender(producers),
		messageStore,
		tx,
		logger,
	)
}

func MustInitSQLMessageStore(
	ctx context.Context,
	sqlClient sql.Client,
) pkgmessage.Store {
	messageStore, err := sql.NewMessageStore(ctx, sqlClient)
	if err != nil {
		panicInitApplication(err)
	}
	return messageStore
}

func MustInitPulsar(optionalLogger log.Logger) pulsar.Connection {
	config := &pulsar.Config{
		Address: env.Must(env.ParseString("PULSAR_ADDRESS")),
	}
	connTimeout, err := env.ParseDuration("PULSAR_CONNECTION_TIMEOUT")
	if err == nil {
		config.ConnectionTimeout = connTimeout
	}

	if optionalLogger == nil {
		optionalLogger = log.NewStub()
	}

	pulsarConn, err := pulsar.NewConnection(config, optionalLogger)
	if err != nil {
		panicInitApplication(err)
	}
	return pulsarConn
}

func MustInitPulsarFailoverConsumer(
	pulsarConn pulsar.Connection,
	topic string,
	subscriptionName string,
) pkgmessage.Consumer {
	consumer, err := pulsarConn.Consumer(&pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscriptionName,
		ConsumptionType:  pulsar.ConsumptionTypeFailover,
	})
	if err != nil {
		panicInitApplication(err)
	}
	return consumer
}

func MustInitPulsarSharedConsumer(
	pulsarConn pulsar.Connection,
	topic string,
	subscriptionName string,
) pkgmessage.Consumer {
	consumer, err := pulsarConn.Consumer(&pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscriptionName,
		ConsumptionType:  pulsar.ConsumptionTypeFailover,
	})
	if err != nil {
		panicInitApplication(err)
	}
	return consumer
}

func panicInitApplication(err error) {
	panic(fmt.Errorf("failed to initialize app: %w", err))
}
