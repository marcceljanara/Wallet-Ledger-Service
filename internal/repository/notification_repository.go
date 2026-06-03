package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository/sqlc"
)

type NotificationRepository interface {
	Create(ctx context.Context, userID uuid.UUID, title, message string) (*model.Notification, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	ClearAll(ctx context.Context, userID uuid.UUID) error
}

type notificationRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *notificationRepository) Create(ctx context.Context, userID uuid.UUID, title, message string) (*model.Notification, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	n, err := r.queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
		UserID:  pgUserID,
		Title:   title,
		Message: message,
	})
	if err != nil {
		return nil, fmt.Errorf("create notification query failed: %w", err)
	}

	return mapNotification(n), nil
}

func (r *notificationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Notification, int, error) {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	notifications, err := r.queries.FindNotificationsByUserID(ctx, sqlc.FindNotificationsByUserIDParams{
		UserID: pgUserID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("find notifications by user id query failed: %w", err)
	}

	total, err := r.queries.CountNotificationsByUserID(ctx, pgUserID)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications by user id query failed: %w", err)
	}

	domainNotifications := make([]model.Notification, len(notifications))
	for i, n := range notifications {
		domainNotifications[i] = *mapNotification(n)
	}

	return domainNotifications, int(total), nil
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	err := r.queries.MarkNotificationAsRead(ctx, sqlc.MarkNotificationAsReadParams{
		ID:     pgID,
		UserID: pgUserID,
	})
	if err != nil {
		return fmt.Errorf("mark notification as read query failed: %w", err)
	}
	return nil
}

func (r *notificationRepository) ClearAll(ctx context.Context, userID uuid.UUID) error {
	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	err := r.queries.ClearNotificationsByUserID(ctx, pgUserID)
	if err != nil {
		return fmt.Errorf("clear notifications query failed: %w", err)
	}
	return nil
}

func mapNotification(n sqlc.Notification) *model.Notification {
	var id uuid.UUID
	if n.ID.Valid {
		id = n.ID.Bytes
	}
	var userID uuid.UUID
	if n.UserID.Valid {
		userID = n.UserID.Bytes
	}
	var createdAt time.Time
	if n.CreatedAt.Valid {
		createdAt = n.CreatedAt.Time
	}

	return &model.Notification{
		ID:        id,
		UserID:    userID,
		Title:     n.Title,
		Message:   n.Message,
		IsRead:    n.IsRead,
		CreatedAt: createdAt,
	}
}
