package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/mocks"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/service"
)

// newDisconnectedRedis returns a *redis.Client that is not connected to any server.
// It is used in service tests to exercise the graceful error-logging path for Redis
// without requiring a running Redis instance.
func newDisconnectedRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:0", // deliberately invalid to ensure connection failure
	})
}

func TestNotificationService_CreateAndPushNotification_Success(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()
	notifID := uuid.New()
	now := time.Now()

	expected := &model.Notification{
		ID:        notifID,
		UserID:    userID,
		Title:     "Top-up Berhasil",
		Message:   "Top-up sebesar Rp 500 berhasil.",
		IsRead:    false,
		CreatedAt: now,
	}

	mockRepo.On("Create", mock.Anything, userID, "Top-up Berhasil", "Top-up sebesar Rp 500 berhasil.").
		Return(expected, nil)

	svc := service.NewNotificationService(mockRepo, redisClient)
	result, err := svc.CreateAndPushNotification(context.Background(), userID, "Top-up Berhasil", "Top-up sebesar Rp 500 berhasil.")

	// Even though Redis publish will fail (disconnected), the DB result should be returned.
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, notifID, result.ID)
	assert.Equal(t, "Top-up Berhasil", result.Title)
}

func TestNotificationService_CreateAndPushNotification_RepoError(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()
	dbErr := errors.New("db connection failed")

	mockRepo.On("Create", mock.Anything, userID, mock.Anything, mock.Anything).
		Return(nil, dbErr)

	svc := service.NewNotificationService(mockRepo, redisClient)
	result, err := svc.CreateAndPushNotification(context.Background(), userID, "Title", "Message")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, dbErr, err)
}

func TestNotificationService_GetNotifications_Success(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()
	now := time.Now()

	notifications := []model.Notification{
		{ID: uuid.New(), UserID: userID, Title: "Login Berhasil", Message: "Anda baru saja masuk.", IsRead: false, CreatedAt: now},
		{ID: uuid.New(), UserID: userID, Title: "Top-up Berhasil", Message: "Top-up Rp 100.", IsRead: true, CreatedAt: now},
	}

	// page=1, limit=10 → offset=0
	mockRepo.On("FindByUserID", mock.Anything, userID, 10, 0).Return(notifications, 2, nil)

	svc := service.NewNotificationService(mockRepo, redisClient)
	result, err := svc.GetNotifications(context.Background(), userID, dto.PaginationRequest{Page: 1, Limit: 10})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Notifications, 2)
	assert.Equal(t, "Login Berhasil", result.Notifications[0].Title)
	assert.Equal(t, 2, result.Pagination.TotalItems)
	assert.Equal(t, 1, result.Pagination.TotalPages)
	assert.Equal(t, 1, result.Pagination.CurrentPage)
	assert.Equal(t, 10, result.Pagination.PerPage)
}

func TestNotificationService_GetNotifications_DefaultPagination(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()

	// When Page and Limit are zero, SetDefaults sets them to Page=1, Limit=10.
	mockRepo.On("FindByUserID", mock.Anything, userID, 10, 0).Return([]model.Notification{}, 0, nil)

	svc := service.NewNotificationService(mockRepo, redisClient)
	result, err := svc.GetNotifications(context.Background(), userID, dto.PaginationRequest{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Notifications, 0)
	assert.Equal(t, 0, result.Pagination.TotalItems)
	// Division by zero safety: when total is 0, totalPages should be 0.
	assert.Equal(t, 0, result.Pagination.TotalPages)
}

func TestNotificationService_GetNotifications_Pagination_MultiPage(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()

	// 25 total items, page 2, limit 10 → offset=10, totalPages=3
	mockRepo.On("FindByUserID", mock.Anything, userID, 10, 10).Return([]model.Notification{}, 25, nil)

	svc := service.NewNotificationService(mockRepo, redisClient)
	result, err := svc.GetNotifications(context.Background(), userID, dto.PaginationRequest{Page: 2, Limit: 10})

	assert.NoError(t, err)
	assert.Equal(t, 25, result.Pagination.TotalItems)
	assert.Equal(t, 3, result.Pagination.TotalPages)
	assert.Equal(t, 2, result.Pagination.CurrentPage)
}

func TestNotificationService_MarkAsRead_Success(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	notifID := uuid.New()
	userID := uuid.New()

	mockRepo.On("MarkAsRead", mock.Anything, notifID, userID).Return(nil)

	svc := service.NewNotificationService(mockRepo, redisClient)
	err := svc.MarkAsRead(context.Background(), notifID, userID)

	assert.NoError(t, err)
}

func TestNotificationService_MarkAsRead_Error(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	notifID := uuid.New()
	userID := uuid.New()
	dbErr := errors.New("notification not found")

	mockRepo.On("MarkAsRead", mock.Anything, notifID, userID).Return(dbErr)

	svc := service.NewNotificationService(mockRepo, redisClient)
	err := svc.MarkAsRead(context.Background(), notifID, userID)

	assert.Error(t, err)
	assert.Equal(t, dbErr, err)
}

func TestNotificationService_ClearAll_Success(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()

	mockRepo.On("ClearAll", mock.Anything, userID).Return(nil)

	svc := service.NewNotificationService(mockRepo, redisClient)
	err := svc.ClearAll(context.Background(), userID)

	assert.NoError(t, err)
}

func TestNotificationService_ClearAll_Error(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	redisClient := newDisconnectedRedis()
	defer redisClient.Close()

	userID := uuid.New()
	dbErr := errors.New("db error")

	mockRepo.On("ClearAll", mock.Anything, userID).Return(dbErr)

	svc := service.NewNotificationService(mockRepo, redisClient)
	err := svc.ClearAll(context.Background(), userID)

	assert.Error(t, err)
	assert.Equal(t, dbErr, err)
}
