package rabbitmq

import (
	"DelayedNotifier/pkg/logger"
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"go.uber.org/zap"
)

type Producer struct {
	publisher *rabbitmq.Publisher
	cfg       *config.Config
}

func NewProducer(cl *ClientRabbitMQ, cfg *config.Config) *Producer {
	publisher := rabbitmq.NewPublisher(cl.client, cfg.GetString("PUBLISHER_EXCHANGE"), "application/json")
	return &Producer{
		publisher: publisher,
		cfg:       cfg,
	}
}

func (p *Producer) Publish(data []byte, ctx context.Context, routingKey string, delay time.Duration) error {
	logger.GetLoggerFromCtx(ctx).Info("Publishing message to RabbitMQ",
		zap.String("routing_key", routingKey),
		zap.Duration("delay", delay))

	err := p.publisher.Publish(
		ctx,
		data,
		routingKey,
		rabbitmq.WithHeaders(amqp.Table{
			"x-delay": delay.Milliseconds(),
		}),
	)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("Failed to publish message",
			zap.Error(err),
			zap.String("routing_key", routingKey))
		return err
	}

	logger.GetLoggerFromCtx(ctx).Info("Message published successfully",
		zap.String("routing_key", routingKey))

	return nil
}
