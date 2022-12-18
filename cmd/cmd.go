package cmd

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/log"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
	"io/fs"
)

const (
	ServiceName = "duck-service"
)

func MustInitSQL(ctx context.Context, logger log.Logger, optionalMigrations fs.ReadDirFS) sql.Connection {
	sqlConfig := &sql.Config{
		DSN: sql.DSN{
			User:     env.MustParseString(ctx, "SQL_USER", logger),
			Password: env.MustParseString(ctx, "SQL_PASSWORD", logger),
			Address:  env.MustParseString(ctx, "SQL_ADDRESS", logger),
			Database: env.MustParseString(ctx, "SQL_DATABASE", logger),
		},
	}
	sqlConnTimeout, ok := env.ParseDuration("SQL_CONNECTION_TIMEOUT")
	if ok {
		sqlConfig.ConnectionTimeout = sqlConnTimeout
	}

	sqlConn, err := sql.NewConnection(sqlConfig, logger)
	if err != nil {
		handleInitApplicationFatal(ctx, logger, err)
	}

	if optionalMigrations == nil {
		return sqlConn
	}
	sqlMigration := sql.NewMigration(sqlConn.Client(), optionalMigrations, logger)
	err = sqlMigration.Execute(ctx)
	if err != nil {
		handleInitApplicationFatal(ctx, logger, err)
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
		handleInitApplicationFatal(ctx, logger, err)
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
	logger log.Logger,
) pkgmessage.Store {
	messageStore, err := sql.NewMessageStore(ctx, sqlClient)
	if err != nil {
		handleInitApplicationFatal(ctx, logger, err)
	}
	return messageStore
}

func MustInitPulsar(ctx context.Context, logger log.Logger) pulsar.Connection {
	config := &pulsar.Config{
		Address: env.MustParseString(ctx, "PULSAR_ADDRESS", logger),
	}
	connTimeout, ok := env.ParseDuration("PULSAR_CONNECTION_TIMEOUT")
	if ok {
		config.ConnectionTimeout = connTimeout
	}

	pulsarConn, err := pulsar.NewConnection(config, logger)
	if err != nil {
		handleInitApplicationFatal(ctx, logger, err)
	}
	return pulsarConn
}

func MustInitPulsarSingleConsumer(
	ctx context.Context,
	pulsarConn pulsar.Connection,
	topic string,
	subscriptionName string,
	logger log.Logger,
) pkgmessage.Consumer {
	consumer, err := pulsarConn.Consumer(&pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscriptionName,
		ConsumptionType:  pulsar.ConsumptionTypeFailover,
	})
	if err != nil {
		handleInitApplicationFatal(ctx, logger, err)
	}
	return consumer
}

func handleInitApplicationFatal(ctx context.Context, logger log.Logger, err error) {
	logger.WithError(err).Fatal(ctx, "failed to initialize app")
}
