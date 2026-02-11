package service

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/internal/rabbitmq"
	"DelayedNotifier/internal/telegram"
	"DelayedNotifier/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/config"
	"go.uber.org/zap"
)

type NotificationRepositoryInterface interface {
	CreateNotification(notification *models.Notification) error
	GetNotification(id string) (*models.Notification, error)
	DeleteNotification(id string) error
	UpdateNotificationStatus(id string, status string) error
}

type DelayedNotifierService struct {
	repo           NotificationRepositoryInterface
	ctx            context.Context
	producer       *rabbitmq.Producer
	cfg            *config.Config
	telegramClient *telegram.Client
}

func New(producer *rabbitmq.Producer, repo NotificationRepositoryInterface, telegramClient *telegram.Client, ctx context.Context, cfg *config.Config) *DelayedNotifierService {
	return &DelayedNotifierService{
		repo:           repo,
		producer:       producer,
		telegramClient: telegramClient,
		ctx:            ctx,
		cfg:            cfg,
	}
}

func (service *DelayedNotifierService) CreateNotification(nf *models.Notification) (string, error) {
	nf.Id = uuid.New().String()

	var delay time.Duration

	if nf.Time == "" {
		delay = 0
		logger.GetLoggerFromCtx(service.ctx).Info("No time specified, sending immediately",
			zap.String("notification_id", nf.Id))
	} else {
		sendTime, err := time.Parse(time.RFC3339, nf.Time)
		if err != nil {
			return "", fmt.Errorf("invalid time format (use RFC3339): %w", err)
		}

		now := time.Now()
		delay = time.Until(sendTime)

		logger.GetLoggerFromCtx(service.ctx).Info("Calculated delay for notification",
			zap.String("notification_id", nf.Id),
			zap.String("current_time", now.Format(time.RFC3339)),
			zap.String("send_time", sendTime.Format(time.RFC3339)),
			zap.Duration("delay_seconds", delay),
			zap.Float64("delay_milliseconds", float64(delay.Milliseconds())))

		if delay < 0 {
			logger.GetLoggerFromCtx(service.ctx).Warn("Requested time is in the past, setting delay to 0",
				zap.String("notification_id", nf.Id),
				zap.String("requested_time", nf.Time),
				zap.Duration("was_negative", delay))
			delay = 0
		}
	}
	data, err := json.Marshal(nf)
	if err != nil {
		return "", fmt.Errorf("failed to marshal notification: %w", err)
	}
	if err = service.producer.Publish(data, service.ctx, service.cfg.GetString("ROUTING_KEY"), delay); err != nil {
		return "", err
	}
	nf.Status = "created"
	err = service.repo.CreateNotification(nf)
	if err != nil {
		return "", err
	}
	return nf.Id, nil
}

func (service *DelayedNotifierService) GetNotification(id string) (*models.Notification, error) {
	if id == "" {
		return nil, errors.New("invalid id")
	}
	nf, err := service.repo.GetNotification(id)
	if err != nil {
		return nil, err
	}
	return nf, nil
}

func (service *DelayedNotifierService) DeleteNotification(id string) error {
	if id == "" {
		return errors.New("invalid id")
	}
	err := service.repo.DeleteNotification(id)
	if err != nil {
		return err
	}
	return nil
}

func (service *DelayedNotifierService) ProcessNotification(nf *models.Notification) error {
	if nf.Id == "" || nf.Message == "" || nf.ChatId == 0 {
		return errors.New("invalid notification: missing required fields")
	}

	if err := service.repo.UpdateNotificationStatus(nf.Id, "sending"); err != nil {
		return fmt.Errorf("failed to update status to sending: %w", err)
	}

	logger.GetLoggerFromCtx(service.ctx).Info("Sending notification via Telegram",
		zap.String("notification_id", nf.Id),
		zap.Int64("chat_id", nf.ChatId),
		zap.String("message", nf.Message))

	if err := service.telegramClient.SendMessage(nf.ChatId, nf.Message); err != nil {
		err = service.repo.UpdateNotificationStatus(nf.Id, "failed")
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	if err := service.repo.UpdateNotificationStatus(nf.Id, "sent"); err != nil {
		return fmt.Errorf("failed to update status to sent: %w", err)
	}

	logger.GetLoggerFromCtx(service.ctx).Info("Notification sent successfully",
		zap.String("notification_id", nf.Id))
	return nil
}
