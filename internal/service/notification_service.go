package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository"
)

type NotificationService interface {
	CreateAndPushNotification(ctx context.Context, userID uuid.UUID, title, message string) (*model.Notification, error)
	GetNotifications(ctx context.Context, userID uuid.UUID, pagination dto.PaginationRequest) (*dto.NotificationListResponse, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ClearAll(ctx context.Context, userID uuid.UUID) error
}

type notificationService struct {
	notificationRepo repository.NotificationRepository
	redisClient      *redis.Client
}

func NewNotificationService(repo repository.NotificationRepository, redisClient *redis.Client) NotificationService {
	return &notificationService{
		notificationRepo: repo,
		redisClient:      redisClient,
	}
}

func (s *notificationService) CreateAndPushNotification(ctx context.Context, userID uuid.UUID, title, message string) (*model.Notification, error) {
	n, err := s.notificationRepo.Create(ctx, userID, title, message)
	if err != nil {
		return nil, err
	}

	// Push notification to Redis Pub/Sub
	channelName := fmt.Sprintf("user:notifications:%s", userID.String())
	payload, err := json.Marshal(n)
	if err == nil {
		_ = s.redisClient.Publish(ctx, channelName, payload).Err()
	}

	return n, nil
}

func (s *notificationService) GetNotifications(ctx context.Context, userID uuid.UUID, pagination dto.PaginationRequest) (*dto.NotificationListResponse, error) {
	pagination.SetDefaults()

	notifications, total, err := s.notificationRepo.FindByUserID(ctx, userID, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	resNotifications := make([]dto.NotificationResponse, len(notifications))
	for i, n := range notifications {
		resNotifications[i] = dto.NotificationResponse{
			ID:        n.ID,
			UserID:    n.UserID,
			Title:     n.Title,
			Message:   n.Message,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt,
		}
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.NotificationListResponse{
		Notifications: resNotifications,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.notificationRepo.MarkAsRead(ctx, id, userID)
}

func (s *notificationService) ClearAll(ctx context.Context, userID uuid.UUID) error {
	return s.notificationRepo.ClearAll(ctx, userID)
}
