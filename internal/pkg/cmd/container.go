package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/klwxsrx/go-service-template/internal/pkg/auth"
	"github.com/klwxsrx/go-service-template/internal/pkg/http"
	pkgauth "github.com/klwxsrx/go-service-template/pkg/auth"
	"github.com/klwxsrx/go-service-template/pkg/env"
	pkghttp "github.com/klwxsrx/go-service-template/pkg/http"
	"github.com/klwxsrx/go-service-template/pkg/idk"
	"github.com/klwxsrx/go-service-template/pkg/lazy"
	"github.com/klwxsrx/go-service-template/pkg/log"
	"github.com/klwxsrx/go-service-template/pkg/message"
	"github.com/klwxsrx/go-service-template/pkg/metric"
	"github.com/klwxsrx/go-service-template/pkg/observability"
	"github.com/klwxsrx/go-service-template/pkg/pulsar"
	"github.com/klwxsrx/go-service-template/pkg/sql"
	pkgtime "github.com/klwxsrx/go-service-template/pkg/time"
)

var logLevelMap = map[string]log.Level{
	"disabled": log.LevelDisabled,
	"debug":    log.LevelDebug,
	"info":     log.LevelInfo,
	"warn":     log.LevelWarn,
	"error":    log.LevelError,
}

type InfrastructureContainer struct {
	HTTPServer             lazy.Loader[pkghttp.Server]
	HTTPClientFactory      lazy.Loader[HTTPClientFactory]
	EventDispatcher        lazy.Loader[message.EventDispatcher]
	TaskScheduler          lazy.Loader[message.TaskScheduler]
	MessageBusListener     lazy.Loader[message.BusListener]
	MessageOutbox          lazy.Loader[message.OutboxProducer]
	IdempotencyKeys        lazy.Loader[idk.Service]
	IdempotencyKeysCleaner lazy.Loader[idk.Cleaner]
	DBMigrations           lazy.Loader[SQLMigrations]
	DB                     lazy.Loader[sql.Database]
	Clock                  lazy.Loader[pkgtime.Clock]
	Metrics                lazy.Loader[metric.Metrics]
	Logger                 lazy.Loader[log.Logger]

	messageBrokerImpl lazy.Loader[*pulsar.MessageBroker]
}

func NewInfrastructureContainer(ctx context.Context) *InfrastructureContainer {
	metrics := metricsProvider()
	logger := loggerProvider()
	observer := observerProvider(logger)
	auth := authProvider()
	clock := clockProvider()

	db := sqlDatabaseProvider(ctx)
	dbMigrations := sqlMigrationsProvider(ctx, db, logger)
	sqlMessageOutboxStorage := sqlMessageOutboxStorageProvider(db, dbMigrations)
	sqlIDKStorage := sqlIDKStorageProvider(db, dbMigrations)

	msgBrokerImpl := pulsarMessageBrokerProvider()
	msgBroker := lazy.New(func() (message.Broker, error) { return msgBrokerImpl.Load() })

	msgBusProducer := messageBusProducerProvider(sqlMessageOutboxStorage, observer, metrics, logger)

	idkServiceImpl := idkServiceProvider(sqlIDKStorage)
	idkService := lazy.New(func() (idk.Service, error) { return idkServiceImpl.Load() })
	idkCleaner := lazy.New(func() (idk.Cleaner, error) { return idkServiceImpl.Load() })

	return &InfrastructureContainer{
		HTTPServer:             httpServerProvider(observer, metrics, logger, auth),
		HTTPClientFactory:      httpClientFactoryProvider(observer, metrics, logger),
		EventDispatcher:        eventDispatcherProvider(msgBusProducer),
		TaskScheduler:          taskSchedulerProvider(msgBusProducer),
		MessageBusListener:     messageBusListenerProvider(msgBroker, observer, metrics, logger),
		MessageOutbox:          messageOutboxProducerProvider(sqlMessageOutboxStorage, msgBroker, metrics, logger),
		IdempotencyKeys:        idkService,
		IdempotencyKeysCleaner: idkCleaner,
		DBMigrations:           dbMigrations,
		DB:                     db,
		Clock:                  clock,
		Metrics:                metrics,
		Logger:                 logger,
		messageBrokerImpl:      msgBrokerImpl,
	}
}

func (i *InfrastructureContainer) Close(ctx context.Context) {
	panicMsg := recover()
	if panicMsg != nil {
		i.Logger.MustLoad().WithField("panic", log.Fields{
			"message": fmt.Sprintf("%v", panicMsg),
			"stack":   string(debug.Stack()),
		}).Error(ctx, "app failed with panic")
		defer os.Exit(1)
	}

	i.MessageOutbox.IfLoaded(func(outbox message.OutboxProducer) { outbox.Close() })
	i.messageBrokerImpl.IfLoaded(func(broker *pulsar.MessageBroker) { broker.Close() })
	i.DB.IfLoaded(func(db sql.Database) {
		if err := db.Close(); err != nil {
			i.Logger.MustLoad().WithError(err).Error(ctx, "failed to close postgresql database")
		}
	})
}

func metricsProvider() lazy.Loader[metric.Metrics] {
	return lazy.New(func() (metric.Metrics, error) {
		return metric.NewMetricsStub(), nil
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
			observability.WithFieldsLogging(logger.MustLoad(), observability.FieldRequestID),
		), nil
	})
}

func authProvider() lazy.Loader[pkgauth.Provider[auth.Principal]] {
	return lazy.New(func() (pkgauth.Provider[auth.Principal], error) {
		return auth.NewProvider(), nil
	})
}

func clockProvider() lazy.Loader[pkgtime.Clock] {
	return lazy.New(func() (pkgtime.Clock, error) {
		return pkgtime.NewAdjustableClock(), nil
	})
}

func sqlDatabaseProvider(ctx context.Context) lazy.Loader[sql.Database] {
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

		db, err := sql.NewDatabase(ctx, sqlConfig)
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
		dbMigrations.MustLoad().MustRegister(sql.MessageOutboxMigrations)
		return sql.NewMessageOutboxStorage(db.MustLoad()), nil
	})
}

func sqlIDKStorageProvider(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[SQLMigrations],
) lazy.Loader[idk.Storage] {
	return lazy.New(func() (idk.Storage, error) {
		dbMigrations.MustLoad().MustRegister(sql.IdempotencyKeyMigrations)
		return sql.NewIdempotencyKeyStorage(db.MustLoad()), nil
	})
}

func httpServerProvider(
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
	auth lazy.Loader[pkgauth.Provider[auth.Principal]],
) lazy.Loader[pkghttp.Server] {
	return lazy.New(func() (pkghttp.Server, error) {
		return pkghttp.NewServer(
			env.Must(env.Parse[string]("SERVICE_ADDRESS")),
			pkghttp.WithHealthCheck(nil),
			pkghttp.WithCORSHandler(),
			pkghttp.WithObservability(observer.MustLoad(), pkghttp.ObservabilityFieldExtractors{
				observability.FieldRequestID: []pkghttp.ObservabilityFieldExtractor{
					pkghttp.ObservabilityFieldHeaderExtractor(http.HeaderRequestID),
					pkghttp.ObservabilityFieldRandomUUIDExtractor(),
				},
			}),
			pkghttp.WithMetrics(metrics.MustLoad()),
			pkghttp.WithLogging(logger.MustLoad(), log.LevelInfo, log.LevelError),
			pkghttp.WithAuth(
				auth.MustLoad(),
				http.UserIDTokenProvider,
				http.AdminUserIDTokenProvider,
				http.ServiceNameTokenProvider,
			),
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
			pkghttp.WithRequestObservability(observer.MustLoad(), pkghttp.ObservabilityFieldHeaders{
				observability.FieldRequestID: http.HeaderRequestID,
			}),
			pkghttp.WithRequestMetrics(metrics.MustLoad()),
			pkghttp.WithRequestLogging(logger.MustLoad(), log.LevelInfo, log.LevelWarn),
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

		messageBroker, err := pulsar.NewMessageBroker(config)
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
			message.WithHandlerObservability(observer.MustLoad(), observability.FieldRequestID),
			message.WithHandlerMetrics(metrics.MustLoad()),
			message.WithHandlerLogging(logger.MustLoad(), log.LevelInfo, log.LevelError),
			message.WithHandlerIdempotencyKeyErrorIgnoring(),
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
			message.WithObservability(observer.MustLoad(), observability.FieldRequestID),
			message.WithMetrics(metrics.MustLoad()),
			message.WithLogging(logger.MustLoad(), log.LevelInfo, log.LevelWarn),
		), nil
	})
}

func idkServiceProvider(idkStorage lazy.Loader[idk.Storage]) lazy.Loader[idk.ServiceImpl] {
	return lazy.New(func() (idk.ServiceImpl, error) {
		return idk.NewService(idkStorage.MustLoad()), nil
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
