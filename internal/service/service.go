package service

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/internal/rabbitmq"
	"DelayedNotifier/internal/telegram"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"
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

func New(producer *rabbitmq.Producer, repo NotificationRepositoryInterface, telegramClient *telegram.Client) *DelayedNotifierService {
	return &DelayedNotifierService{
		repo:           repo,
		producer:       producer,
		telegramClient: telegramClient,
	}
}

func (service *DelayedNotifierService) CreateNotification(nf *models.Notification) (string, error) {
	nf.Id = uuid.New().String()
	sendTime, err := time.Parse(time.RFC3339, nf.Time)
	if err != nil {
		return "", fmt.Errorf("invalid time format: %w", err)
	}
	delay := time.Until(sendTime)
	if delay < 0 {
		delay = 0
	}
	data, err := json.Marshal(nf)
	if err != nil {
		return "", fmt.Errorf("failed to marshal notification: %w", err)
	}
	if err = service.producer.Publish(data, service.ctx, service.cfg.GetString("routing_key"), delay); err != nil {
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

	zlog.Logger.Info().
		Str("notification_id", nf.Id).
		Int64("chat_id", nf.ChatId).
		Str("message", nf.Message).
		Msg("Sending notification via Telegram")

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

	zlog.Logger.Info().
		Str("notification_id", nf.Id).
		Msg("Notification sent successfully")
	return nil
}
