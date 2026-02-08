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
	publisher := rabbitmq.NewPublisher(cl.client, cfg.GetString("publisher_exchange"), "application/json")
	return &Producer{
		publisher: publisher,
		cfg:       cfg,
	}
}

func (p *Producer) Publish(data []byte, ctx context.Context, routingKey string) error {
	err := p.publisher.Publish(
		ctx,
		data,
		routingKey,
		rabbitmq.WithExpiration(5*time.Minute),
		rabbitmq.WithHeaders(amqp.Table{"x-service": "auth"}),
	)
	if err != nil {
		zlog.Logger.Info().Err(err).Str("routing_key", routingKey).Msg("publish fail")
		return err
	}
	return nil
}
