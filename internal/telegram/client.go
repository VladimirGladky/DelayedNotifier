package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"
)

type Client struct {
	bot *tgbotapi.BotAPI
	cfg *config.Config
}

func NewClient(cfg *config.Config) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.GetString("telegram_bot_token"))
	if err != nil {
		return nil, err
	}

	zlog.Logger.Info().
		Str("bot_username", bot.Self.UserName).
		Msg("Telegram bot authorized successfully")

	return &Client{
		bot: bot,
		cfg: cfg,
	}, nil
}

func (c *Client) SendMessage(chatId int64, message string) error {
	msg := tgbotapi.NewMessage(chatId, message)

	_, err := c.bot.Send(msg)
	if err != nil {
		zlog.Logger.Error().
			Err(err).
			Int64("chat_id", chatId).
			Msg("Failed to send telegram message")
		return err
	}

	zlog.Logger.Info().
		Int64("chat_id", chatId).
		Msg("Telegram message sent successfully")

	return nil
}
