package rabbitmq

import (
	"DelayedNotifier/internal/models"
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/zlog"
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
		zlog.Logger.Debug().
			Str("queue", cfg.GetString("CONSUMER_QUEUE")).
			Str("message_id", d.MessageId).
			Msg("Received message from queue")

		var notification models.Notification
		if err := json.Unmarshal(d.Body, &notification); err != nil {
			zlog.Logger.Error().
				Err(err).
				Str("body", string(d.Body)).
				Msg("Failed to unmarshal notification")
			return err
		}

		if err := handler(&notification); err != nil {
			zlog.Logger.Error().
				Err(err).
				Str("notification_id", notification.Id).
				Msg("Failed to process notification")
			return err
		}

		zlog.Logger.Info().
			Str("notification_id", notification.Id).
			Msg("Notification processed successfully")

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
