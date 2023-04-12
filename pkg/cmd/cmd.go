package cmd

import (
	"context"
	"fmt"
	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/log"
	pkglogstub "github.com/klwxsrx/go-service-template/pkg/log/stub"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
	"io/fs"
)

func HandleAppPanic(ctx context.Context, logger log.Logger) {
	msg := recover()
	if msg == nil {
		return
	}

	err, ok := msg.(error)
	if ok {
		logger = logger.WithError(err)
	} else {
		logger = logger.WithField("error", msg)
	}
	logger.Fatal(ctx, "app failed with panic")
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
		panic(fmt.Errorf("open sql connection: %w", err))
	}

	if optionalMigrations == nil {
		return sqlConn
	}
	sqlMigration := sql.NewMigration(sqlConn.Client(), optionalMigrations, logger)
	err = sqlMigration.Execute(ctx)
	if err != nil {
		panic(fmt.Errorf("execute migrations: %w", err))
	}
	return sqlConn
}

func MustInitSQLTransaction(
	sqlClient sql.TxClient,
	instanceName string,
	onCommit func(),
) (sql.Client, persistence.Transaction) {
	return sql.NewTransaction(sqlClient, instanceName, onCommit)
}

func MustInitSQLMessageOutbox(
	ctx context.Context,
	sqlClient sql.TxClient,
	msgProducer pkgmessage.Producer,
	logger log.Logger,
) pkgmessage.Outbox {
	wrappedSQLClient, tx := MustInitSQLTransaction(sqlClient, "messageOutbox", func() {})
	messageStore, err := sql.NewMessageStore(ctx, wrappedSQLClient)
	if err != nil {
		panic(fmt.Errorf("init message outbox: %w", err))
	}
	return pkgmessage.NewOutbox(
		msgProducer,
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
		panic(fmt.Errorf("init message store: %w", err))
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
		optionalLogger = pkglogstub.NewLogger()
	}

	pulsarConn, err := pulsar.NewConnection(config, optionalLogger)
	if err != nil {
		panic(fmt.Errorf("open pulsar connection: %w", err))
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
		panic(fmt.Errorf("init pulsar failover consumer %s/%s: %w", subscriptionName, topic, err))
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
		ConsumptionType:  pulsar.ConsumptionTypeShared,
	})
	if err != nil {
		panic(fmt.Errorf("init pulsar shared consumer %s/%s: %w", subscriptionName, topic, err))
	}
	return consumer
}
