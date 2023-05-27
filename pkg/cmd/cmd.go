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
	"runtime/debug"
)

func HandleAppPanic(ctx context.Context, logger log.Logger) {
	msg := recover()
	if msg == nil {
		return
	}

	err, ok := msg.(error)
	if ok {
		msg = err.Error()
	}

	logger.WithField("panic", log.Fields{
		"message": msg,
		"stack":   string(debug.Stack()),
	}).Fatal(ctx, "app failed with panic")
}

func MustInitSQL(ctx context.Context, logger log.Logger, optionalMigrations fs.ReadDirFS) sql.Database {
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

	if optionalMigrations == nil {
		return db
	}
	sqlMigration := sql.NewMigration(db, optionalMigrations, logger)
	err = sqlMigration.Execute(ctx)
	if err != nil {
		panic(fmt.Errorf("execute migrations: %w", err))
	}
	return db
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
	messageDispatcher pkgmessage.Dispatcher,
	logger log.Logger,
) pkgmessage.Outbox {
	wrappedSQLClient, tx := MustInitSQLTransaction(sqlClient, "messageOutbox", func() {})
	messageStore, err := sql.NewMessageStore(ctx, wrappedSQLClient)
	if err != nil {
		panic(fmt.Errorf("init message outbox: %w", err))
	}
	return pkgmessage.NewOutbox(
		messageDispatcher,
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

func MustInitPulsarMessageBroker(optionalLogger log.Logger) *pulsar.MessageBroker {
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

	messageBroker, err := pulsar.NewMessageBroker(config, optionalLogger)
	if err != nil {
		panic(fmt.Errorf("open pulsar connection: %w", err))
	}
	return messageBroker
}
