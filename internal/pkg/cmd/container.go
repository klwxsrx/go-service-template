package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	internalauth "github.com/klwxsrx/go-service-template/internal/pkg/auth"
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
	HTTPServer              lazy.Loader[pkghttp.Server]
	HTTPClientFactory       lazy.Loader[HTTPClientFactory]
	EventDispatcher         lazy.Loader[message.EventDispatcher]
	TaskScheduler           lazy.Loader[message.TaskScheduler]
	MessageBusListener      lazy.Loader[message.BusListener]
	MessageStorageConsumers lazy.Loader[message.StorageConsumerProvider]
	IdempotencyKeys         lazy.Loader[idk.Service]
	IdempotencyKeysCleaner  lazy.Loader[idk.Cleaner]
	DBMigrations            lazy.Loader[SQLMigrations]
	DB                      lazy.Loader[sql.Database]
	Clock                   lazy.Loader[pkgtime.Clock]
	Observer                lazy.Loader[observability.Observer]
	Metrics                 lazy.Loader[metric.Metrics]
	Logger                  lazy.Loader[log.Logger]
}

func NewInfrastructureContainer(ctx context.Context) *InfrastructureContainer {
	metrics := metricsProvider()
	logger := loggerProvider()
	observer := observerProvider(logger)
	auth := authProvider()
	clock := clockProvider()

	db := sqlDatabaseProvider(ctx)
	dbMigrations := sqlMigrationsProvider(ctx, db, logger)
	msgStorage := sqlMessageStorageProvider(db, dbMigrations)
	idkStorage := sqlIDKStorageProvider(db, dbMigrations)

	msgStorageConsumerProvider := messageStorageConsumerProvider(msgStorage, metrics, logger)
	consumerProvider := lazy.New(func() (message.ConsumerProvider[message.AckStrategy], error) {
		return msgStorageConsumerProvider.MustLoad(), nil
	})

	msgBusProducer := messageBusProducerProvider(msgStorage, observer, metrics, logger)
	idkServiceImpl := idkServiceProvider(idkStorage)
	idkService := lazy.New(func() (idk.Service, error) { return idkServiceImpl.Load() })
	idkCleaner := lazy.New(func() (idk.Cleaner, error) { return idkServiceImpl.Load() })

	return &InfrastructureContainer{
		HTTPServer:              httpServerProvider(observer, metrics, logger, auth),
		HTTPClientFactory:       httpClientFactoryProvider(observer, metrics, logger),
		EventDispatcher:         eventDispatcherProvider(msgBusProducer),
		TaskScheduler:           taskSchedulerProvider(msgBusProducer),
		MessageBusListener:      messageBusListenerProvider(consumerProvider, observer, metrics, logger),
		MessageStorageConsumers: msgStorageConsumerProvider,
		IdempotencyKeys:         idkService,
		IdempotencyKeysCleaner:  idkCleaner,
		DBMigrations:            dbMigrations,
		DB:                      db,
		Clock:                   clock,
		Observer:                observer,
		Metrics:                 metrics,
		Logger:                  logger,
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

func authProvider() lazy.Loader[pkgauth.Provider[internalauth.Principal]] {
	return lazy.New(func() (pkgauth.Provider[internalauth.Principal], error) {
		return internalauth.NewProvider(), nil
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

func sqlMessageStorageProvider(
	db lazy.Loader[sql.Database],
	dbMigrations lazy.Loader[SQLMigrations],
) lazy.Loader[message.Storage] {
	return lazy.New(func() (message.Storage, error) {
		dbMigrations.MustLoad().MustRegister(sql.MessageStorageMigrations)
		return sql.NewMessageStorage(db.MustLoad()), nil
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
	auth lazy.Loader[pkgauth.Provider[internalauth.Principal]],
) lazy.Loader[pkghttp.Server] {
	return lazy.New(func() (pkghttp.Server, error) {
		return pkghttp.NewServer(
			pkghttp.WithServerAddress(env.Must(env.Parse[string]("SERVICE_ADDRESS"))),
			pkghttp.WithHandlerOptions(
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

func messageBusListenerProvider(
	msgConsumers lazy.Loader[message.ConsumerProvider[message.AckStrategy]],
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[message.BusListener] {
	return lazy.New(func() (message.BusListener, error) {
		return message.NewBusListener(
			msgConsumers.MustLoad(),
			message.NewAckQueue,
			func() message.Deserializer { return message.NewJSONSerializer() },
			message.WithHandlerObservability(observer.MustLoad(), observability.FieldRequestID),
			message.WithHandlerMetrics(metrics.MustLoad()),
			message.WithHandlerLogging(logger.MustLoad(), log.LevelInfo, log.LevelError),
			message.WithHandlerIdempotencyKeyErrorIgnoring(),
		), nil
	})
}

func messageStorageConsumerProvider(
	msgStorage lazy.Loader[message.Storage],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[message.StorageConsumerProvider] {
	return lazy.New(func() (message.StorageConsumerProvider, error) {
		return message.NewStorageConsumerProvider(
			msgStorage.MustLoad(),
			message.WithStorageConsumerMetrics(metrics.MustLoad()),
			message.WithStorageConsumerLogging(logger.MustLoad(), log.LevelDebug, log.LevelWarn),
		), nil
	})
}

func messageBusProducerProvider(
	msgStorage lazy.Loader[message.Storage],
	observer lazy.Loader[observability.Observer],
	metrics lazy.Loader[metric.Metrics],
	logger lazy.Loader[log.Logger],
) lazy.Loader[message.BusScheduledProducer] {
	return lazy.New(func() (message.BusScheduledProducer, error) {
		return message.NewBusScheduledProducer(
			msgStorage.MustLoad(),
			message.NewJSONSerializer(),
			message.WithBusProducerObservability(observer.MustLoad(), observability.FieldRequestID),
			message.WithBusProducerMetrics(metrics.MustLoad()),
			message.WithBusProducerLogging(logger.MustLoad(), log.LevelDebug, log.LevelWarn),
		), nil
	})
}

func idkServiceProvider(idkStorage lazy.Loader[idk.Storage]) lazy.Loader[idk.ServiceImpl] {
	return lazy.New(func() (idk.ServiceImpl, error) {
		return idk.NewService(idkStorage.MustLoad()), nil
	})
}

func eventDispatcherProvider(busProducer lazy.Loader[message.BusScheduledProducer]) lazy.Loader[message.EventDispatcher] {
	return lazy.New(func() (message.EventDispatcher, error) {
		return message.NewEventDispatcher(busProducer.MustLoad()), nil
	})
}

func taskSchedulerProvider(busProducer lazy.Loader[message.BusScheduledProducer]) lazy.Loader[message.TaskScheduler] {
	return lazy.New(func() (message.TaskScheduler, error) {
		return message.NewTaskScheduler(busProducer.MustLoad()), nil
	})
}
