package rabbitmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/zlog"
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
	zlog.Logger.Info().
		Str("routing_key", routingKey).
		Dur("delay", delay).
		Msg("Publishing message to RabbitMQ")

	err := p.publisher.Publish(
		ctx,
		data,
		routingKey,
		rabbitmq.WithHeaders(amqp.Table{
			"x-delay": delay.Milliseconds(),
		}),
	)
	if err != nil {
		zlog.Logger.Error().
			Err(err).
			Str("routing_key", routingKey).
			Msg("Failed to publish message")
		return err
	}

	zlog.Logger.Info().
		Str("routing_key", routingKey).
		Msg("Message published successfully")

	return nil
}
