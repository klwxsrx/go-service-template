package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	commonhttp "github.com/klwxsrx/go-service-template/internal/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/cmd"
	"github.com/klwxsrx/go-service-template/pkg/env"
	"github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/lazy"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	pkgmetricstub "github.com/klwxsrx/go-service-template/pkg/metric/stub"
	"github.com/klwxsrx/go-service-template/pkg/observability"
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

type InfrastructureContainer struct {
	HTTPServer         lazy.Loader[http.Server]
	HTTPClientFactory  lazy.Loader[HTTPClientFactory]
	EventDispatcher    lazy.Loader[message.EventDispatcher]
	TaskScheduler      lazy.Loader[message.TaskScheduler]
	MessageBusListener lazy.Loader[message.BusListener]
	MessageOutbox      lazy.Loader[message.OutboxProducer]
	DBMigrations       lazy.Loader[SQLMigrations]
	DB                 lazy.Loader[sql.Database]
	Metrics            lazy.Loader[metric.Metrics]
	Logger             lazy.Loader[log.Logger]

	messageBrokerImpl lazy.Loader[*pulsar.MessageBroker]
}

func NewInfrastructureContainer(ctx context.Context) *InfrastructureContainer {
	metrics := metricsProvider()
	logger := loggerProvider()
	observer := observerProvider(logger)

	db := sqlDatabaseProvider(ctx, logger)
	dbMigrations := sqlMigrationsProvider(ctx, db, logger)
	sqlMessageOutboxStorage := sqlMessageOutboxStorageProvider(db, dbMigrations)

	msgBrokerImpl := pulsarMessageBrokerProvider()
	msgBroker := lazy.New(func() (message.Broker, error) { return msgBrokerImpl.Load() })

	msgBusProducer := messageBusProducerProvider(sqlMessageOutboxStorage, observer, metrics, logger)

	return &InfrastructureContainer{
		HTTPServer:         httpServerProvider(observer, metrics, logger),
		HTTPClientFactory:  httpClientFactoryProvider(observer, metrics, logger),
		EventDispatcher:    eventDispatcherProvider(msgBusProducer),
		TaskScheduler:      taskSchedulerProvider(msgBusProducer),
		MessageBusListener: messageBusListenerProvider(msgBroker, observer, metrics, logger),
		MessageOutbox:      messageOutboxProducerProvider(sqlMessageOutboxStorage, msgBroker, metrics, logger),
		DBMigrations:       dbMigrations,
		DB:                 db,
		Metrics:            metrics,
		Logger:             logger,
		messageBrokerImpl:  msgBrokerImpl,
	}
}

func (i *InfrastructureContainer) Close(ctx context.Context) {
	if cmd.HandleAppPanic(ctx, i.Logger.MustLoad()) {
		defer os.Exit(1)
	}

	i.MessageOutbox.IfLoaded(func(outbox message.OutboxProducer) { outbox.Close() })
	i.messageBrokerImpl.IfLoaded(func(broker *pulsar.MessageBroker) { broker.Close() })
	i.DB.IfLoaded(func(db sql.Database) { db.Close(ctx) })
}

func metricsProvider() lazy.Loader[metric.Metrics] {
	return lazy.New(func() (metric.Metrics, error) {
		return pkgmetricstub.NewMetrics(), nil
	})
}

func loggerProvider() lazy.Loader[log.Logger] {
	return lazy.New(func() (log.Logger, error) {
		logLevelStr, err := env.Parse[string]("LOG_LEVEL")
		if err != nil {
			return log.New(log.LevelInfo), nil
		}

		logLevel, ok := logLevelMap[logLevelStr]
		if !ok {
			logLevel = log.LevelInfo
		}

		return log.New(logLevel), nil
	})
}

func observerProvider(
	logger lazy.Loader[log.Logger],
) lazy.Loader[observability.Observer] {
	return lazy.New(func() (observability.Observer, error) {
		return observability.New(
			observability.WithFieldsLogging(logger.MustLoad(), observability.LogFieldRequestID),
		), nil
	})
}

func sqlDatabaseProvider(
	ctx context.Context,
	logger lazy.Loader[log.Logger],
) lazy.Loader[sql.Database] {
	return lazy.New(func() (sql.Database, error) {
		sqlConfig := &sql.Config{
			DSN: sql.DSN{
				User:     env.Must(env.Parse[string]("SQL_USER")),
				Password: env.Must(env.Parse[string]("SQL_PASSWORD")),
				Address:  env.Must(env.Parse[string]("SQL_ADDRESS")),
				Database: env.Must(env.Parse[string]("SQL_DATABASE")),
			},
			MaxOpenConnections: env.Must(env.Parse[int]("SQL_MAX_OPEN_CONNECTIONS")),
			MaxIdleConnections: env.Must(env.Parse[int]("SQL_MAX_IDLE_CONNECTIONS")),
		}
		sqlConnTimeout := env.Must(env.ParseOptional[*time.Duration]("SQL_CONNECTION_TIMEOUT"))
		if sqlConnTimeout != nil {
			sqlConfig.ConnectionTimeout = *sqlConnTimeout
		}

		db, err := sql.NewDatabase(ctx, sqlConfig, logger.MustLoad())
		if err != nil {
			panic(fmt.Errorf("open sql connection: %w", err))
		}

		return db, nil
	})
}

func sqlMigrationsProvider(
	ctx context.Context,
	db lazy.Loader[sql.Database],
	logger lazy.Loader[log.Logger],
) lazy.Loader[SQLMigrations] {
	return lazy.New(func() (SQLMigrations, error) {
		return NewSQLMigrations(ctx, db.MustLoad(), logger.MustLoad()), nil
	})
}

func sqlMessageOutboxStorageProvider(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[SQLMigrations],
) lazy.Loader[message.OutboxStorage] {
	return lazy.New(func() (message.OutboxStorage, error) {
		dbMigrations.MustLoad().MustRegisterSource(sql.MessageOutboxMigrations)
		return sql.NewMessageOutboxStorage(db.MustLoad()), nil
	})
}

func httpServerProvider(
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[http.Server] {
	return lazy.New(func() (http.Server, error) {
		return http.NewServer(
			http.DefaultServerAddress,
			http.WithHealthCheck(nil),
			http.WithCORSHandler(),
			http.WithObservability(
				observer.MustLoad(),
				http.NewHTTPHeaderRequestIDExtractor(commonhttp.RequestIDHeader),
				http.NewRandomUUIDRequestIDExtractor(),
			),
			http.WithMetrics(metrics.MustLoad()),
			http.WithLogging(logger.MustLoad(), log.LevelInfo, log.LevelError),
		), nil
	})
}

func httpClientFactoryProvider(
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[HTTPClientFactory] {
	return lazy.New(func() (HTTPClientFactory, error) {
		return NewHTTPClientFactory(
			http.WithRequestObservability(observer.MustLoad(), commonhttp.RequestIDHeader),
			http.WithRequestMetrics(metrics.MustLoad()),
			http.WithRequestLogging(logger.MustLoad(), log.LevelInfo, log.LevelWarn),
		), nil
	})
}

func pulsarMessageBrokerProvider() lazy.Loader[*pulsar.MessageBroker] {
	return lazy.New(func() (*pulsar.MessageBroker, error) {
		config := &pulsar.Config{
			Address: env.Must(env.Parse[string]("PULSAR_ADDRESS")),
		}
		connTimeout := env.Must(env.ParseOptional[*time.Duration]("PULSAR_CONNECTION_TIMEOUT"))
		if connTimeout != nil {
			config.ConnectionTimeout = *connTimeout
		}

		stubLogger := log.New(log.LevelDisabled)
		messageBroker, err := pulsar.NewMessageBroker(config, stubLogger)
		if err != nil {
			panic(fmt.Errorf("open pulsar connection: %w", err))
		}

		return messageBroker, nil
	})
}

func messageBusListenerProvider(
	msgBroker lazy.Loader[message.Broker],
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[message.BusListener] {
	return lazy.New(func() (message.BusListener, error) {
		return message.NewBusListener(
			msgBroker.MustLoad(),
			message.WithHandlerObservability(observer.MustLoad()),
			message.WithHandlerMetrics(metrics.MustLoad()),
			message.WithHandlerLogging(logger.MustLoad(), log.LevelInfo, log.LevelError),
		), nil
	})
}

func messageBusProducerProvider(
	outboxStorage lazy.Loader[message.OutboxStorage],
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[message.BusProducer] {
	return lazy.New(func() (message.BusProducer, error) {
		return message.NewBusProducer(
			outboxStorage.MustLoad(),
			message.WithObservability(observer.MustLoad()),
			message.WithMetrics(metrics.MustLoad()),
			message.WithLogging(logger.MustLoad(), log.LevelInfo, log.LevelWarn),
		), nil
	})
}

func eventDispatcherProvider(busProducer lazy.Loader[message.BusProducer]) lazy.Loader[message.EventDispatcher] {
	return lazy.New(func() (message.EventDispatcher, error) {
		return message.NewEventDispatcher(busProducer.MustLoad()), nil
	})
}

func taskSchedulerProvider(busProducer lazy.Loader[message.BusProducer]) lazy.Loader[message.TaskScheduler] {
	return lazy.New(func() (message.TaskScheduler, error) {
		return message.NewTaskScheduler(busProducer.MustLoad()), nil
	})
}

func messageOutboxProducerProvider(
	outboxStorage lazy.Loader[message.OutboxStorage],
	msgBroker lazy.Loader[message.Broker],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[message.OutboxProducer] {
	return lazy.New(func() (message.OutboxProducer, error) {
		return message.NewOutboxProducer(
			outboxStorage.MustLoad(),
			msgBroker.MustLoad(),
			message.WithOutboxProducerMetrics(metrics.MustLoad()),
			message.WithOutboxProducerLogging(logger.MustLoad(), log.LevelInfo, log.LevelWarn),
		), nil
	})
}
