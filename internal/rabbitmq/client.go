package rabbitmq

import (
	"context"
	"time"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

type ClientRabbitMQ struct {
	cfg    *config.Config
	ctx    *context.Context
	client *rabbitmq.RabbitClient
}

func NewClientRabbitMQ(cfg *config.Config, ctx *context.Context) *ClientRabbitMQ {
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
		URL:            c.cfg.GetString("rabbitmq_url"),
		ConnectionName: c.cfg.GetString("rabbitmq_connection_name"),
		ConnectTimeout: time.Duration(c.cfg.GetInt("connect_timeout")) * time.Second,
		Heartbeat:      time.Duration(c.cfg.GetInt("Heartbeat")) * time.Second,
		ProducingStrat: strategy,
		ConsumingStrat: strategy,
	}

	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return err
	}
	c.client = client
	defer client.Close()
	return nil
}
