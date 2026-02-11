package rabbitmq

import (
	"DelayedNotifier/pkg/logger"
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	"go.uber.org/zap"
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
		logger.GetLoggerFromCtx(c.ctx).Error("Failed to declare delayed exchange", zap.Error(err), zap.String("exchange", delayedExchange))
		return err
	}
	logger.GetLoggerFromCtx(c.ctx).Info("Delayed exchange declared", zap.String("exchange", delayedExchange))

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
		logger.GetLoggerFromCtx(c.ctx).Error("Failed to declare DLX exchange", zap.Error(err), zap.String("exchange", dlxExchange))
		return err
	}
	logger.GetLoggerFromCtx(c.ctx).Info("DLX exchange declared", zap.String("exchange", dlxExchange))

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
		logger.GetLoggerFromCtx(c.ctx).Error("Failed to declare main queue", zap.Error(err), zap.String("queue", mainQueue))
		return err
	}
	logger.GetLoggerFromCtx(c.ctx).Info("Main queue declared and bound",
		zap.String("queue", mainQueue),
		zap.String("exchange", delayedExchange),
		zap.String("routing_key", routingKey))

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
		logger.GetLoggerFromCtx(c.ctx).Error("Failed to declare DLQ", zap.Error(err), zap.String("queue", dlqQueue))
		return err
	}
	logger.GetLoggerFromCtx(c.ctx).Info("DLQ declared and bound",
		zap.String("queue", dlqQueue),
		zap.String("exchange", dlxExchange),
		zap.String("routing_key", dlqRoutingKey))

	logger.GetLoggerFromCtx(c.ctx).Info("RabbitMQ infrastructure setup completed successfully")
	return nil
}
