package rabbitmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
)

type ClientRabbitMQ struct {
	cfg    *config.Config
	ctx    context.Context
	client *rabbitmq.RabbitClient
}

func NewClientRabbitMQ(cfg *config.Config, ctx context.Context) *ClientRabbitMQ {
	return &ClientRabbitMQ{
		cfg: cfg,
		ctx: ctx,
	}
}

func (c *ClientRabbitMQ) Init() error {
	strategy := retry.Strategy{
		Attempts: 3,
		Delay:    3 * time.Second,
		Backoff:  2,
	}

	cfg := rabbitmq.ClientConfig{
		URL:            c.cfg.GetString("RABBITMQ_URL"),
		ConnectionName: c.cfg.GetString("RABBITMQ_CONNECTION_NAME"),
		ConnectTimeout: time.Duration(c.cfg.GetInt("CONNECT_TIMEOUT")) * time.Second,
		Heartbeat:      time.Duration(c.cfg.GetInt("HEARTBEAT")) * time.Second,
		ProducingStrat: strategy,
		ConsumingStrat: strategy,
	}

	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

func (c *ClientRabbitMQ) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

func (c *ClientRabbitMQ) SetupInfrastructure() error {
	delayedExchange := c.cfg.GetString("PUBLISHER_EXCHANGE")
	err := c.client.DeclareExchange(
		delayedExchange,
		"x-delayed-message",
		true,
		false,
		false,
		amqp.Table{
			"x-delayed-type": "direct",
		},
	)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("exchange", delayedExchange).Msg("Failed to declare delayed exchange")
		return err
	}
	zlog.Logger.Info().Str("exchange", delayedExchange).Msg("Delayed exchange declared")

	dlxExchange := c.cfg.GetString("DLX_EXCHANGE")
	err = c.client.DeclareExchange(
		dlxExchange,
		"direct",
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("exchange", dlxExchange).Msg("Failed to declare DLX exchange")
		return err
	}
	zlog.Logger.Info().Str("exchange", dlxExchange).Msg("DLX exchange declared")

	mainQueue := c.cfg.GetString("CONSUMER_QUEUE")
	routingKey := c.cfg.GetString("ROUTING_KEY")
	dlqRoutingKey := c.cfg.GetString("DLQ_ROUTING_KEY")

	err = c.client.DeclareQueue(
		mainQueue,
		delayedExchange,
		routingKey,
		true,
		false,
		true,
		amqp.Table{
			"x-dead-letter-exchange":    dlxExchange,
			"x-dead-letter-routing-key": dlqRoutingKey,
		},
	)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("queue", mainQueue).Msg("Failed to declare main queue")
		return err
	}
	zlog.Logger.Info().
		Str("queue", mainQueue).
		Str("exchange", delayedExchange).
		Str("routing_key", routingKey).
		Msg("Main queue declared and bound")

	dlqQueue := "dlq_queue"
	err = c.client.DeclareQueue(
		dlqQueue,
		dlxExchange,
		dlqRoutingKey,
		true,
		false,
		true,
		nil,
	)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("queue", dlqQueue).Msg("Failed to declare DLQ")
		return err
	}
	zlog.Logger.Info().
		Str("queue", dlqQueue).
		Str("exchange", dlxExchange).
		Str("routing_key", dlqRoutingKey).
		Msg("DLQ declared and bound")

	zlog.Logger.Info().Msg("RabbitMQ infrastructure setup completed successfully")
	return nil
}
