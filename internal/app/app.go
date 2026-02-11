package app

import (
	"DelayedNotifier/internal/rabbitmq"
	"DelayedNotifier/internal/repository"
	"DelayedNotifier/internal/service"
	"DelayedNotifier/internal/telegram"
	"DelayedNotifier/internal/transport"
	"DelayedNotifier/pkg/postgres"
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"
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

	telegramClient, err := telegram.NewClient(cfg)
	if err != nil {
		panic(err)
	}

	producer := rabbitmq.NewProducer(rabbitMQClient, cfg)
	srv := service.New(producer, repo, telegramClient, ctx, cfg)
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
		zlog.Logger.Info().Str("service", "http_server").Msg("Starting HTTP server")
		if err := a.HiTalentServer.Run(); err != nil {
			zlog.Logger.Error().Err(err).Msg("HTTP server error")
			errCh <- err
		}
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		zlog.Logger.Info().Str("service", "rabbitmq_consumer").Msg("Starting RabbitMQ consumer")
		a.rabbitmqConsumer.Start(a.ctx)
		zlog.Logger.Info().Str("service", "rabbitmq_consumer").Msg("Consumer stopped")
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		zlog.Logger.Error().Err(err).Msg("Application error, shutting down")
		a.cancel()
	case sig := <-sigCh:
		zlog.Logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		a.cancel()
	}

	zlog.Logger.Info().Msg("Waiting for goroutines to finish")
	a.wg.Wait()

	zlog.Logger.Info().Msg("Closing connections")
	a.rabbitmqClient.Close()

	zlog.Logger.Info().Msg("Application stopped gracefully")
	return nil
}
