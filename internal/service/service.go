package service

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/internal/rabbitmq"
	"DelayedNotifier/internal/telegram"
	"DelayedNotifier/pkg/logger"
	"DelayedNotifier/pkg/redis"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/config"
	wbfredis "github.com/wb-go/wbf/redis"
	"go.uber.org/zap"
)

type NotificationRepositoryInterface interface {
	CreateNotification(notification *models.Notification) error
	GetNotificationStatus(id string) (string, error)
	DeleteNotification(id string) error
	UpdateNotificationStatus(id string, status string) error
	GetAllNotifications() ([]*models.Notification, error)
}

type DelayedNotifierService struct {
	repo           NotificationRepositoryInterface
	ctx            context.Context
	producer       *rabbitmq.Producer
	cfg            *config.Config
	telegramClient *telegram.Client
	redis          *wbfredis.Client
}

func New(producer *rabbitmq.Producer, repo NotificationRepositoryInterface, telegramClient *telegram.Client, redisClient *wbfredis.Client, ctx context.Context, cfg *config.Config) *DelayedNotifierService {
	return &DelayedNotifierService{
		repo:           repo,
		producer:       producer,
		telegramClient: telegramClient,
		redis:          redisClient,
		ctx:            ctx,
		cfg:            cfg,
	}
}

func (service *DelayedNotifierService) CreateNotification(nf *models.Notification) (string, error) {
	nf.Id = uuid.New().String()

	var delay time.Duration

	if nf.Time == "" {
		delay = 0
	} else {
		sendTime, err := time.Parse(time.RFC3339, nf.Time)
		if err != nil {
			return "", fmt.Errorf("invalid time format (use RFC3339): %w", err)
		}
		delay = time.Until(sendTime)
		if delay < 0 {
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

	logger.GetLoggerFromCtx(service.ctx).Info("Attempting to cache notification status",
		zap.String("notification_id", nf.Id),
		zap.String("status", nf.Status),
		zap.String("key", redis.CacheKey(nf.Id)))

	if err = service.redis.SetWithExpiration(service.ctx, redis.CacheKey(nf.Id), nf.Status, redis.StatusCacheTTL); err != nil {
		logger.GetLoggerFromCtx(service.ctx).Error("Failed to cache notification status",
			zap.Error(err),
			zap.String("notification_id", nf.Id))
	} else {
		logger.GetLoggerFromCtx(service.ctx).Info("Successfully cached notification status",
			zap.String("notification_id", nf.Id))
	}

	return nf.Id, nil
}

func (service *DelayedNotifierService) GetNotificationStatus(id string) (string, error) {
	if id == "" {
		return "", errors.New("invalid id")
	}

	status, err := service.redis.Get(service.ctx, redis.CacheKey(id))
	if err == nil {
		logger.GetLoggerFromCtx(service.ctx).Debug("Notification status retrieved from cache",
			zap.String("notification_id", id),
			zap.String("status", status))
		return status, nil
	}

	status, err = service.repo.GetNotificationStatus(id)
	if err != nil {
		return "", err
	}

	if err := service.redis.SetWithExpiration(service.ctx, redis.CacheKey(id), status, redis.StatusCacheTTL); err != nil {
		logger.GetLoggerFromCtx(service.ctx).Warn("Failed to cache notification status after DB read",
			zap.Error(err),
			zap.String("notification_id", id))
	}

	return status, nil
}

func (service *DelayedNotifierService) DeleteNotification(id string) error {
	if id == "" {
		return errors.New("invalid id")
	}
	err := service.repo.DeleteNotification(id)
	if err != nil {
		return err
	}

	if err := service.redis.Del(service.ctx, redis.CacheKey(id)); err != nil {
		logger.GetLoggerFromCtx(service.ctx).Warn("Failed to delete status from cache",
			zap.Error(err),
			zap.String("notification_id", id))
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

	if err := service.redis.SetWithExpiration(service.ctx, redis.CacheKey(nf.Id), "sending", redis.StatusCacheTTL); err != nil {
		logger.GetLoggerFromCtx(service.ctx).Warn("Failed to update status in cache",
			zap.Error(err),
			zap.String("notification_id", nf.Id))
	}

	logger.GetLoggerFromCtx(service.ctx).Info("Sending notification to Telegram",
		zap.String("notification_id", nf.Id),
		zap.Int64("chat_id", nf.ChatId),
		zap.String("message", nf.Message))

	err := service.telegramClient.SendMessage(nf.ChatId, nf.Message)
	if err != nil {
		logger.GetLoggerFromCtx(service.ctx).Error("Failed to send telegram message",
			zap.Error(err),
			zap.String("notification_id", nf.Id))

		if updateErr := service.repo.UpdateNotificationStatus(nf.Id, "failed"); updateErr != nil {
			logger.GetLoggerFromCtx(service.ctx).Error("Failed to update notification status to failed",
				zap.Error(updateErr))
			return updateErr
		}

		if err := service.redis.SetWithExpiration(service.ctx, redis.CacheKey(nf.Id), "failed", redis.StatusCacheTTL); err != nil {
			logger.GetLoggerFromCtx(service.ctx).Warn("Failed to update status in cache",
				zap.Error(err),
				zap.String("notification_id", nf.Id))
		}

		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	logger.GetLoggerFromCtx(service.ctx).Info("Telegram send succeeded",
		zap.String("notification_id", nf.Id))

	if err = service.repo.UpdateNotificationStatus(nf.Id, "sent"); err != nil {
		return fmt.Errorf("failed to update status to sent: %w", err)
	}

	if err := service.redis.SetWithExpiration(service.ctx, redis.CacheKey(nf.Id), "sent", redis.StatusCacheTTL); err != nil {
		logger.GetLoggerFromCtx(service.ctx).Warn("Failed to update status in cache",
			zap.Error(err),
			zap.String("notification_id", nf.Id))
	}

	logger.GetLoggerFromCtx(service.ctx).Info("Notification sent successfully",
		zap.String("notification_id", nf.Id))
	return nil
}

func (service *DelayedNotifierService) GetAllNotifications() ([]*models.Notification, error) {
	return service.repo.GetAllNotifications()
}
