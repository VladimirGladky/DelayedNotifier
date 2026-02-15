package service

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/internal/repository/mocks"
	servicemocks "DelayedNotifier/internal/service/mocks"
	"DelayedNotifier/pkg/logger"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/config"
	"go.uber.org/mock/gomock"
)

func setupTestContext() context.Context {
	ctx := context.Background()
	ctx, err := logger.New(ctx)
	if err != nil {
		panic(err)
	}
	return ctx
}

func TestDelayedNotifierService_CreateNotificationSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	producer := servicemocks.NewMockRabbitMQProducerInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	inputNotification := &models.Notification{
		Message: "Test notification",
		Time:    "2026-02-13T15:00:00+03:00",
		ChatId:  123456789,
	}

	producer.EXPECT().Publish(gomock.Any(), gomock.Any(), "test.routing.key", gomock.Any()).Return(nil).Times(1)
	repo.EXPECT().CreateNotification(gomock.Any()).Return(nil).Times(1)
	redisClient.EXPECT().SetWithExpiration(gomock.Any(), gomock.Any(), "created", gomock.Any()).Return(nil).Times(1)

	ctx := setupTestContext()
	cfg := config.New()
	cfg.EnableEnv("")
	cfg.SetDefault("ROUTING_KEY", "test.routing.key")

	srv := &DelayedNotifierService{
		repo:     repo,
		producer: producer,
		redis:    redisClient,
		ctx:      ctx,
		cfg:      cfg,
	}

	id, err := srv.CreateNotification(inputNotification)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	require.Equal(t, id, inputNotification.Id)
	require.Equal(t, "created", inputNotification.Status)
}

func TestDelayedNotifierService_CreateNotificationInvalidTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	producer := servicemocks.NewMockRabbitMQProducerInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	inputNotification := &models.Notification{
		Message: "Test notification",
		Time:    "invalid-time-format",
		ChatId:  123456789,
	}

	ctx := setupTestContext()
	cfg := &config.Config{}

	srv := &DelayedNotifierService{
		repo:     repo,
		producer: producer,
		redis:    redisClient,
		ctx:      ctx,
		cfg:      cfg,
	}

	id, err := srv.CreateNotification(inputNotification)
	require.Error(t, err)
	require.Empty(t, id)
	require.Contains(t, err.Error(), "invalid time format")
}

func TestDelayedNotifierService_CreateNotificationPublishError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	producer := servicemocks.NewMockRabbitMQProducerInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	inputNotification := &models.Notification{
		Message: "Test notification",
		Time:    "",
		ChatId:  123456789,
	}

	expectedErr := errors.New("rabbitmq publish error")
	producer.EXPECT().Publish(gomock.Any(), gomock.Any(), "test.routing.key", gomock.Any()).Return(expectedErr).Times(1)

	ctx := setupTestContext()
	cfg := config.New()
	cfg.EnableEnv("")
	cfg.SetDefault("ROUTING_KEY", "test.routing.key")

	srv := &DelayedNotifierService{
		repo:     repo,
		producer: producer,
		redis:    redisClient,
		ctx:      ctx,
		cfg:      cfg,
	}

	id, err := srv.CreateNotification(inputNotification)
	require.Error(t, err)
	require.Empty(t, id)
	require.Equal(t, expectedErr, err)
}

func TestDelayedNotifierService_GetNotificationStatusFromCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	notifID := "test-id-123"
	expectedStatus := "created"

	redisClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedStatus, nil).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:  repo,
		redis: redisClient,
		ctx:   ctx,
	}

	status, err := srv.GetNotificationStatus(notifID)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, status)
}

func TestDelayedNotifierService_GetNotificationStatusFromDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	notifID := "test-id-123"
	expectedStatus := "sent"

	redisClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return("", errors.New("cache miss")).Times(1)
	repo.EXPECT().GetNotificationStatus(notifID).Return(expectedStatus, nil).Times(1)
	redisClient.EXPECT().SetWithExpiration(gomock.Any(), gomock.Any(), expectedStatus, gomock.Any()).Return(nil).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:  repo,
		redis: redisClient,
		ctx:   ctx,
	}

	status, err := srv.GetNotificationStatus(notifID)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, status)
}

func TestDelayedNotifierService_GetNotificationStatusValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:  repo,
		redis: redisClient,
		ctx:   ctx,
	}

	status, err := srv.GetNotificationStatus("")
	require.Error(t, err)
	require.Empty(t, status)
	require.Contains(t, err.Error(), "invalid id")
}

func TestDelayedNotifierService_DeleteNotificationSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)
	notifID := "test-id-123"

	repo.EXPECT().DeleteNotification(notifID).Return(nil).Times(1)
	redisClient.EXPECT().Del(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:  repo,
		redis: redisClient,
		ctx:   ctx,
	}

	err := srv.DeleteNotification(notifID)
	require.NoError(t, err)
}

func TestDelayedNotifierService_DeleteNotificationFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:  repo,
		redis: redisClient,
		ctx:   ctx,
	}

	err := srv.DeleteNotification("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid id")
}

func TestDelayedNotifierService_DeleteNotificationRepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)
	notifID := "test-id-123"
	expectedErr := errors.New("database error")

	repo.EXPECT().DeleteNotification(notifID).Return(expectedErr).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:  repo,
		redis: redisClient,
		ctx:   ctx,
	}

	err := srv.DeleteNotification(notifID)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestDelayedNotifierService_GetAllNotificationsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	expectedNotifications := []*models.Notification{
		{
			Id:      "id-1",
			Message: "Test message 1",
			Time:    "2026-02-13T15:00:00+03:00",
			Status:  "created",
			ChatId:  123456789,
		},
		{
			Id:      "id-2",
			Message: "Test message 2",
			Time:    "2026-02-14T15:00:00+03:00",
			Status:  "sent",
			ChatId:  987654321,
		},
	}

	repo.EXPECT().GetAllNotifications().Return(expectedNotifications, nil).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo: repo,
		ctx:  ctx,
	}

	notifications, err := srv.GetAllNotifications()
	require.NoError(t, err)
	require.Equal(t, expectedNotifications, notifications)
	require.Len(t, notifications, 2)
}

func TestDelayedNotifierService_GetAllNotificationsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	expectedErr := errors.New("database connection error")

	repo.EXPECT().GetAllNotifications().Return(nil, expectedErr).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo: repo,
		ctx:  ctx,
	}

	notifications, err := srv.GetAllNotifications()
	require.Error(t, err)
	require.Nil(t, notifications)
	require.Equal(t, expectedErr, err)
}

func TestDelayedNotifierService_ProcessNotificationSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	telegramClient := servicemocks.NewMockTelegramClientInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	notification := &models.Notification{
		Id:      "test-id",
		Message: "Test message",
		ChatId:  123456789,
	}

	repo.EXPECT().UpdateNotificationStatus("test-id", "sending").Return(nil).Times(1)
	redisClient.EXPECT().SetWithExpiration(gomock.Any(), gomock.Any(), "sending", gomock.Any()).Return(nil).Times(1)
	telegramClient.EXPECT().SendMessage(int64(123456789), "Test message").Return(nil).Times(1)
	repo.EXPECT().UpdateNotificationStatus("test-id", "sent").Return(nil).Times(1)
	redisClient.EXPECT().SetWithExpiration(gomock.Any(), gomock.Any(), "sent", gomock.Any()).Return(nil).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:           repo,
		telegramClient: telegramClient,
		redis:          redisClient,
		ctx:            ctx,
	}

	err := srv.ProcessNotification(notification)
	require.NoError(t, err)
}

func TestDelayedNotifierService_ProcessNotificationTelegramError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)
	telegramClient := servicemocks.NewMockTelegramClientInterface(ctrl)
	redisClient := servicemocks.NewMockRedisClientInterface(ctrl)

	notification := &models.Notification{
		Id:      "test-id",
		Message: "Test message",
		ChatId:  123456789,
	}

	telegramErr := errors.New("telegram api error")

	repo.EXPECT().UpdateNotificationStatus("test-id", "sending").Return(nil).Times(1)
	redisClient.EXPECT().SetWithExpiration(gomock.Any(), gomock.Any(), "sending", gomock.Any()).Return(nil).Times(1)
	telegramClient.EXPECT().SendMessage(int64(123456789), "Test message").Return(telegramErr).Times(1)
	repo.EXPECT().UpdateNotificationStatus("test-id", "failed").Return(nil).Times(1)
	redisClient.EXPECT().SetWithExpiration(gomock.Any(), gomock.Any(), "failed", gomock.Any()).Return(nil).Times(1)

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo:           repo,
		telegramClient: telegramClient,
		redis:          redisClient,
		ctx:            ctx,
	}

	err := srv.ProcessNotification(notification)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to send telegram message")
}

func TestDelayedNotifierService_ProcessNotificationValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockNotificationRepositoryInterface(ctrl)

	cases := []struct {
		name         string
		notification *models.Notification
		expErr       string
	}{
		{
			name: "missing id",
			notification: &models.Notification{
				Id:      "",
				Message: "Test message",
				ChatId:  123456789,
			},
			expErr: "missing required fields",
		},
		{
			name: "missing message",
			notification: &models.Notification{
				Id:      "test-id",
				Message: "",
				ChatId:  123456789,
			},
			expErr: "missing required fields",
		},
		{
			name: "missing chat_id",
			notification: &models.Notification{
				Id:      "test-id",
				Message: "Test message",
				ChatId:  0,
			},
			expErr: "missing required fields",
		},
	}

	ctx := setupTestContext()
	srv := &DelayedNotifierService{
		repo: repo,
		ctx:  ctx,
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := srv.ProcessNotification(tc.notification)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expErr)
		})
	}
}
