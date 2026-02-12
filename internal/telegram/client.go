package telegram

import (
	"DelayedNotifier/pkg/logger"
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/wb-go/wbf/config"
	"go.uber.org/zap"
)

type Client struct {
	bot *tgbotapi.BotAPI
	cfg *config.Config
	ctx context.Context
}

func NewClient(cfg *config.Config, ctx context.Context) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.GetString("telegram_bot_token"))
	if err != nil {
		return nil, err
	}

	logger.GetLoggerFromCtx(ctx).Info("Telegram bot authorized successfully",
		zap.String("bot_username", bot.Self.UserName))

	return &Client{
		bot: bot,
		cfg: cfg,
		ctx: ctx,
	}, nil
}

func (c *Client) SendMessage(chatId int64, message string) error {
	msg := tgbotapi.NewMessage(chatId, message)

	_, err := c.bot.Send(msg)
	if err != nil {
		return err
	}

	logger.GetLoggerFromCtx(c.ctx).Info("Telegram message sent successfully",
		zap.Int64("chat_id", chatId))

	return nil
}
