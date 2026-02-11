package rabbitmq

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/pkg/logger"
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"go.uber.org/zap"
)

type Consumer struct {
	consumer *rabbitmq.Consumer
	cfg      *config.Config
}

func NewConsumer(client *ClientRabbitMQ, cfg *config.Config, handler func(*models.Notification) error) *Consumer {
	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    cfg.GetString("DLX_EXCHANGE"),
		"x-dead-letter-routing-key": cfg.GetString("DLQ_ROUTING_KEY"),
	}

	consumerCfg := rabbitmq.ConsumerConfig{
		Queue: cfg.GetString("CONSUMER_QUEUE"),
		Args:  queueArgs,
	}

	amqpHandler := func(ctx context.Context, d amqp.Delivery) error {
		logger.GetLoggerFromCtx(ctx).Debug("Received message from queue",
			zap.String("queue", cfg.GetString("CONSUMER_QUEUE")),
			zap.String("message_id", d.MessageId))

		var notification models.Notification
		if err := json.Unmarshal(d.Body, &notification); err != nil {
			logger.GetLoggerFromCtx(ctx).Error("Failed to unmarshal notification",
				zap.Error(err),
				zap.String("body", string(d.Body)))
			return err
		}

		if err := handler(&notification); err != nil {
			logger.GetLoggerFromCtx(ctx).Error("Failed to process notification",
				zap.Error(err),
				zap.String("notification_id", notification.Id))
			return err
		}

		logger.GetLoggerFromCtx(ctx).Info("Notification processed successfully",
			zap.String("notification_id", notification.Id))

		return nil
	}

	consumer := rabbitmq.NewConsumer(client.client, consumerCfg, amqpHandler)

	return &Consumer{
		consumer: consumer,
		cfg:      cfg,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		if err := c.consumer.Start(ctx); err != nil {
			log.Fatalf("Ошибка при потреблении сообщений: %v", err)
		}
		done <- struct{}{}
	}()
	<-done
}
