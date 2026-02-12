package app

import (
	"DelayedNotifier/internal/migrations"
	"DelayedNotifier/internal/rabbitmq"
	"DelayedNotifier/internal/repository"
	"DelayedNotifier/internal/service"
	"DelayedNotifier/internal/telegram"
	"DelayedNotifier/internal/transport"
	"DelayedNotifier/pkg/logger"
	"DelayedNotifier/pkg/postgres"
	redispkg "DelayedNotifier/pkg/redis"
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/wb-go/wbf/config"
	"go.uber.org/zap"
)

type App struct {
	HiTalentServer   *transport.Server
	cfg              *config.Config
	ctx              context.Context
	wg               sync.WaitGroup
	cancel           context.CancelFunc
	rabbitmqConsumer *rabbitmq.Consumer
	rabbitmqClient   *rabbitmq.ClientRabbitMQ
}

func NewApp(cfg *config.Config, parentCtx context.Context) *App {
	ctx, cancel := context.WithCancel(parentCtx)

	db, err := postgres.NewPostgres(cfg)
	if err != nil {
		panic(err)
	}

	applied, err := migrations.RunMigrations(db.Master, "./migrations")
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("Failed to run migrations", zap.Error(err))
		panic(err)
	}
	if applied {
		logger.GetLoggerFromCtx(ctx).Info("Database migrations completed successfully")
	}

	redisClient, err := redispkg.NewRedisClient(cfg, ctx)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("Failed to connect to Redis", zap.Error(err))
		panic(err)
	}
	logger.GetLoggerFromCtx(ctx).Info("Connected to Redis successfully")

	repo := repository.NewNotificationRepository(ctx, db)
	rabbitMQClient := rabbitmq.NewClientRabbitMQ(cfg, ctx)
	err = rabbitMQClient.Init()
	if err != nil {
		panic(err)
	}

	err = rabbitMQClient.SetupInfrastructure()
	if err != nil {
		panic(err)
	}

	telegramClient, err := telegram.NewClient(cfg, ctx)
	if err != nil {
		panic(err)
	}

	producer := rabbitmq.NewProducer(rabbitMQClient, cfg)
	srv := service.New(producer, repo, telegramClient, redisClient, ctx, cfg)
	consumer := rabbitmq.NewConsumer(rabbitMQClient, cfg, srv.ProcessNotification)
	server := transport.NewServer(ctx, cfg, srv)

	return &App{
		HiTalentServer:   server,
		cfg:              cfg,
		ctx:              ctx,
		cancel:           cancel,
		rabbitmqConsumer: consumer,
		rabbitmqClient:   rabbitMQClient,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

func (a *App) Run() error {
	errCh := make(chan error, 1)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		logger.GetLoggerFromCtx(a.ctx).Info("starting server")
		if err := a.HiTalentServer.Run(); err != nil {
			logger.GetLoggerFromCtx(a.ctx).Error("HTTP server error", zap.Error(err))
			errCh <- err
		}
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		logger.GetLoggerFromCtx(a.ctx).Info("Starting RabbitMQ consumer", zap.String("service", "rabbitmq_consumer"))
		a.rabbitmqConsumer.Start(a.ctx)
		logger.GetLoggerFromCtx(a.ctx).Info("Consumer stopped", zap.String("service", "rabbitmq_consumer"))
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.GetLoggerFromCtx(a.ctx).Error("Application error, shutting down", zap.Error(err))
		a.cancel()
	case sig := <-sigCh:
		logger.GetLoggerFromCtx(a.ctx).Info("Received shutdown signal", zap.String("signal", sig.String()))
		a.cancel()
	}

	logger.GetLoggerFromCtx(a.ctx).Info("Waiting for goroutines to finish")
	a.wg.Wait()

	logger.GetLoggerFromCtx(a.ctx).Info("Closing connections")
	a.rabbitmqClient.Close()

	logger.GetLoggerFromCtx(a.ctx).Info("Application stopped gracefully")
	return nil
}
