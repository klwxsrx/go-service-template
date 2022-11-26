package cmd

import (
	"context"
	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/log"
	pkgmessage "github.com/klwxsrx/go-service-template/pkg/message"
	pkgpersistence "github.com/klwxsrx/go-service-template/pkg/persistence"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
	"io/fs"
)

const (
	ServiceName = "duck-service"
)

func MustInitSQL(ctx context.Context, logger log.Logger, optionalMigrations fs.ReadDirFS) sql.Connection {
	sqlConn, err := sql.NewConnection(&sql.Config{
		DSN: sql.DSN{
			User:     env.MustGetString(ctx, "SQL_USER", logger),
			Password: env.MustGetString(ctx, "SQL_PASSWORD", logger),
			Address:  env.MustGetString(ctx, "SQL_ADDRESS", logger),
			Database: env.MustGetString(ctx, "SQL_DATABASE", logger),
		},
	}, logger)
	if err != nil {
		logger.Fatal(ctx, err.Error())
	}

	if optionalMigrations == nil {
		return sqlConn
	}
	sqlMigration := sql.NewMigration(sqlConn.Client(), optionalMigrations, logger)
	err = sqlMigration.Execute(ctx)
	if err != nil {
		logger.Fatal(ctx, err.Error())
	}
	return sqlConn
}

func MustInitSQLMessageOutbox(
	sqlConn sql.Connection,
	producers pkgmessage.ProducerProvider,
	logger log.Logger,
) pkgpersistence.MessageOutbox {
	return pkgpersistence.NewMessageOutbox(
		pkgmessage.NewSender(producers),
		sql.NewMessageStore(sqlConn.Client()),
		sql.NewCriticalSection(sqlConn.Client()),
		logger,
	)
}

func MustInitPulsar(ctx context.Context, logger log.Logger) pulsar.Connection {
	pulsarConn, err := pulsar.NewConnection(&pulsar.Config{
		Address:   env.MustGetString(ctx, "PULSAR_ADDRESS", logger),
		AuthToken: env.MustGetString(ctx, "PULSAR_AUTH_TOKEN", logger),
	}, logger)
	if err != nil {
		logger.Fatal(ctx, err.Error())
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
		logger.Fatalf(ctx, "failed to init topic %s consumer by %s subscriber", topic, subscriptionName)
	}
	return consumer
}
