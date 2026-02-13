package repository

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/pkg/logger"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wb-go/wbf/dbpg"
	"go.uber.org/zap"
)

type NotificationRepository struct {
	ctx context.Context
	db  *dbpg.DB
}

func NewNotificationRepository(ctx context.Context, db *dbpg.DB) *NotificationRepository {
	return &NotificationRepository{
		ctx: ctx,
		db:  db,
	}
}

func (r *NotificationRepository) CreateNotification(notification *models.Notification) error {
	query := `
		INSERT INTO notifications (id, message, time, status, chat_id)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(
		r.ctx,
		query,
		notification.Id,
		notification.Message,
		notification.Time,
		notification.Status,
		notification.ChatId,
	)
	if err != nil {
		logger.GetLoggerFromCtx(r.ctx).Error("Failed to create notification in DB",
			zap.Error(err),
			zap.String("notification_id", notification.Id))
		return err
	}

	logger.GetLoggerFromCtx(r.ctx).Info("Notification created in DB",
		zap.String("notification_id", notification.Id))
	return nil
}

func (r *NotificationRepository) GetNotificationStatus(id string) (string, error) {
	query := `
  		SELECT status
  		FROM notifications
  		WHERE id = $1
  	`

	var status string

	err := r.db.QueryRowContext(r.ctx, query, id).Scan(
		&status,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("notification not found: %s", id)
		}
		logger.GetLoggerFromCtx(r.ctx).Error("Failed to get notification from DB",
			zap.Error(err),
			zap.String("notification_id", id))
		return "", fmt.Errorf("failed to get notification: %w", err)
	}

	return status, nil
}

func (r *NotificationRepository) UpdateNotificationStatus(id string, status string) error {
	query := `
  		UPDATE notifications
  		SET status = $1
  		WHERE id = $2
  	`
	_, err := r.db.ExecContext(r.ctx, query, status, id)
	if err != nil {
		logger.GetLoggerFromCtx(r.ctx).Error("Failed to update notification status",
			zap.Error(err),
			zap.String("notification_id", id),
			zap.String("status", status))
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	logger.GetLoggerFromCtx(r.ctx).Debug("Notification status updated",
		zap.String("notification_id", id),
		zap.String("status", status))
	return nil
}

func (r *NotificationRepository) DeleteNotification(id string) error {
	query := `
  		DELETE FROM notifications
  		WHERE id = $1
  	`

	result, err := r.db.ExecContext(r.ctx, query, id)
	if err != nil {
		logger.GetLoggerFromCtx(r.ctx).Error("Failed to delete notification",
			zap.Error(err),
			zap.String("notification_id", id))
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("notification not found: %s", id)
	}

	logger.GetLoggerFromCtx(r.ctx).Info("Notification deleted from DB",
		zap.String("notification_id", id))
	return nil
}

func (r *NotificationRepository) GetAllNotifications() ([]*models.Notification, error) {
	query := `
		SELECT id, message, time, status, chat_id
		FROM notifications
	`

	rows, err := r.db.QueryContext(r.ctx, query)
	if err != nil {
		logger.GetLoggerFromCtx(r.ctx).Error("Failed to get all notifications",
			zap.Error(err))
		return nil, fmt.Errorf("failed to get all notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		nf := &models.Notification{}
		err := rows.Scan(&nf.Id, &nf.Message, &nf.Time, &nf.Status, &nf.ChatId)
		if err != nil {
			logger.GetLoggerFromCtx(r.ctx).Error("Failed to scan notification",
				zap.Error(err))
			continue
		}
		notifications = append(notifications, nf)
	}

	return notifications, nil
}
