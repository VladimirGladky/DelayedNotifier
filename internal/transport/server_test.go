package transport

import (
	"DelayedNotifier/internal/models"
	"DelayedNotifier/internal/service/mocks"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/config"
	"go.uber.org/mock/gomock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNotifyCreateHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	expectedID := "test-id-123"

	inputNotification := &models.Notification{
		Message: "Test notification",
		Time:    "2026-02-13T15:00:00+03:00",
		ChatId:  123456789,
	}

	srv.EXPECT().CreateNotification(gomock.Any()).Return(expectedID, nil).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.POST("/api/v1/notify", server.NotifyCreateHandler())

	body, _ := json.Marshal(inputNotification)
	req := httptest.NewRequest("POST", "/api/v1/notify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, expectedID, response["id"])
}

func TestNotifyCreateHandler_Fail(t *testing.T) {
	cases := []struct {
		name           string
		requestBody    string
		expectedStatus int
		setupMock      func(*mocks.MockServiceDelayedNotifierInterface)
	}{
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(m *mocks.MockServiceDelayedNotifierInterface) {},
		},
		{
			name:           "empty body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(m *mocks.MockServiceDelayedNotifierInterface) {},
		},
		{
			name:           "malformed JSON",
			requestBody:    `{"message": }`,
			expectedStatus: http.StatusBadRequest,
			setupMock:      func(m *mocks.MockServiceDelayedNotifierInterface) {},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
			tc.setupMock(srv)

			cfg := &config.Config{}
			ctx := context.Background()
			server := NewServer(ctx, cfg, srv)

			router := gin.New()
			router.POST("/api/v1/notify", server.NotifyCreateHandler())

			req := httptest.NewRequest("POST", "/api/v1/notify", bytes.NewBufferString(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestNotifyCreateHandler_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	expectedErr := errors.New("service error")

	inputNotification := &models.Notification{
		Message: "Test notification",
		Time:    "2026-02-13T15:00:00+03:00",
		ChatId:  123456789,
	}

	srv.EXPECT().CreateNotification(gomock.Any()).Return("", expectedErr).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.POST("/api/v1/notify", server.NotifyCreateHandler())

	body, _ := json.Marshal(inputNotification)
	req := httptest.NewRequest("POST", "/api/v1/notify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNotifyGetHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	notifID := "test-id-123"
	expectedStatus := "created"

	srv.EXPECT().GetNotificationStatus(notifID).Return(expectedStatus, nil).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.GET("/api/v1/notify/:id", server.NotifyGetHandler())

	req := httptest.NewRequest("GET", "/api/v1/notify/"+notifID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, response["status"])
}

func TestNotifyGetHandler_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	notifID := "test-id-123"
	expectedErr := errors.New("notification not found")

	srv.EXPECT().GetNotificationStatus(notifID).Return("", expectedErr).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.GET("/api/v1/notify/:id", server.NotifyGetHandler())

	req := httptest.NewRequest("GET", "/api/v1/notify/"+notifID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNotifyDeleteHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	notifID := "test-id-123"

	srv.EXPECT().DeleteNotification(notifID).Return(nil).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.DELETE("/api/v1/notify/:id", server.NotifyDeleteHandler())

	req := httptest.NewRequest("DELETE", "/api/v1/notify/"+notifID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Contains(t, response["status"], notifID)
	require.Contains(t, response["status"], "deleted")
}

func TestNotifyDeleteHandler_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	notifID := "test-id-123"
	expectedErr := errors.New("notification not found")

	srv.EXPECT().DeleteNotification(notifID).Return(expectedErr).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.DELETE("/api/v1/notify/:id", server.NotifyDeleteHandler())

	req := httptest.NewRequest("DELETE", "/api/v1/notify/"+notifID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAllNotificationsHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
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

	srv.EXPECT().GetAllNotifications().Return(expectedNotifications, nil).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.GET("/api/v1/notifications", server.GetAllNotificationsHandler())

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response []*models.Notification
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response, 2)
	require.Equal(t, expectedNotifications[0].Id, response[0].Id)
	require.Equal(t, expectedNotifications[1].Id, response[1].Id)
}

func TestGetAllNotificationsHandler_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := mocks.NewMockServiceDelayedNotifierInterface(ctrl)
	expectedErr := errors.New("database error")

	srv.EXPECT().GetAllNotifications().Return(nil, expectedErr).Times(1)

	cfg := &config.Config{}
	ctx := context.Background()
	server := NewServer(ctx, cfg, srv)

	router := gin.New()
	router.GET("/api/v1/notifications", server.GetAllNotificationsHandler())

	req := httptest.NewRequest("GET", "/api/v1/notifications", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
