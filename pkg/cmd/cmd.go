package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
)

var logLevelMap = map[string]log.Level{
	"disabled": log.LevelDisabled,
	"debug":    log.LevelDebug,
	"info":     log.LevelInfo,
	"warn":     log.LevelWarn,
	"error":    log.LevelError,
}

func HandleAppPanic(ctx context.Context, logger log.Logger) {
	msg := recover()
	if msg == nil {
		return
	}

	logger.WithField("panic", log.Fields{
		"message": fmt.Sprintf("%v", msg),
		"stack":   string(debug.Stack()),
	}).Error(ctx, "app failed with panic")
	os.Exit(1)
}

func InitLogger() log.Logger {
	logLevelStr, err := env.ParseString("LOG_LEVEL")
	if err != nil {
		return log.New(log.LevelInfo)
	}

	logLevel, ok := logLevelMap[logLevelStr]
	if !ok {
		logLevel = log.LevelInfo
	}
	return log.New(logLevel)
}

func InitHTTPClientFactory(
	opts ...http.ClientOption,
) HTTPClientFactory {
	return HTTPClientFactory{
		impl: http.NewClientFactory(opts...),
	}
}

func MustInitSQL(ctx context.Context, logger log.Logger, migrations ...sql.MigrationSource) sql.Database {
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

	db, err := sql.NewDatabase(sqlConfig, logger)
	if err != nil {
		panic(fmt.Errorf("open sql connection: %w", err))
	}

	if len(migrations) == 0 {
		return db
	}
	err = sql.NewMigrator(db, logger).Execute(ctx, migrations...)
	if err != nil {
		panic(fmt.Errorf("execute migrations: %w", err))
	}

	return db
}

func MustInitSQLMessageOutbox(
	sqlClient sql.TxClient,
	messageProducer message.Producer,
	logger log.Logger,
) message.Outbox {
	wrappedSQLClient, tx := sql.NewTransaction(sqlClient, "messageOutbox", func() {})
	messageStorage := sql.NewMessageOutboxStorage(wrappedSQLClient)

	return message.NewOutbox(
		messageStorage,
		messageProducer,
		tx,
		logger,
	)
}

func MustInitPulsarMessageBroker(optionalLogger log.Logger) *pulsar.MessageBroker {
	config := &pulsar.Config{
		Address: env.Must(env.ParseString("PULSAR_ADDRESS")),
	}
	connTimeout, err := env.ParseDuration("PULSAR_CONNECTION_TIMEOUT")
	if err == nil {
		config.ConnectionTimeout = connTimeout
	}

	if optionalLogger == nil {
		optionalLogger = log.New(log.LevelDisabled)
	}

	messageBroker, err := pulsar.NewMessageBroker(config, optionalLogger)
	if err != nil {
		panic(fmt.Errorf("open pulsar connection: %w", err))
	}
	return messageBroker
}
