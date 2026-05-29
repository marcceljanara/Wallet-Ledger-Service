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

type AuditRepository interface {
	Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.AuditLog, int, error)
	FindAll(ctx context.Context, userID *uuid.UUID, action *string, limit, offset int) ([]model.AuditLog, int, error)
}

type auditRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewAuditRepository(pool *pgxpool.Pool) AuditRepository {
	return &auditRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *auditRepository) Create(ctx context.Context, log *model.AuditLog) (*model.AuditLog, error) {
	pgUUID := pgtype.UUID{Bytes: log.UserID, Valid: true}

	al, err := r.queries.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		ID:        log.ID,
		UserID:    pgUUID,
		Action:    log.Action,
		IpAddress: log.IPAddress,
		Endpoint:  log.Endpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("create audit log query failed: %w", err)
	}
	return mapAuditLog(al), nil
}

func (r *auditRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.AuditLog, int, error) {
	pgUUID := pgtype.UUID{Bytes: userID, Valid: true}
	logs, err := r.queries.FindAuditLogsByUserID(ctx, sqlc.FindAuditLogsByUserIDParams{
		UserID: pgUUID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("find audit logs by user id query failed: %w", err)
	}

	total, err := r.queries.CountAuditLogsByUserID(ctx, pgUUID)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit logs by user id query failed: %w", err)
	}

	domainLogs := make([]model.AuditLog, len(logs))
	for i, l := range logs {
		domainLogs[i] = *mapAuditLog(l)
	}
	return domainLogs, int(total), nil
}

func (r *auditRepository) FindAll(ctx context.Context, userID *uuid.UUID, action *string, limit, offset int) ([]model.AuditLog, int, error) {
	var logs []sqlc.AuditLog
	var err error
	var total int64

	if userID != nil && action != nil {
		pgUUID := pgtype.UUID{Bytes: *userID, Valid: true}
		logs, err = r.queries.FindAllAuditLogsByUserIDAndAction(ctx, sqlc.FindAllAuditLogsByUserIDAndActionParams{
			UserID: pgUUID,
			Action: *action,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all audit logs by user id and action query failed: %w", err)
		}

		total, err = r.queries.CountAllAuditLogsByUserIDAndAction(ctx, sqlc.CountAllAuditLogsByUserIDAndActionParams{
			UserID: pgUUID,
			Action: *action,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("count all audit logs by user id and action query failed: %w", err)
		}
	} else if userID != nil {
		pgUUID := pgtype.UUID{Bytes: *userID, Valid: true}
		logs, err = r.queries.FindAllAuditLogsByUserID(ctx, sqlc.FindAllAuditLogsByUserIDParams{
			UserID: pgUUID,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all audit logs by user id query failed: %w", err)
		}

		total, err = r.queries.CountAllAuditLogsByUserID(ctx, pgUUID)
		if err != nil {
			return nil, 0, fmt.Errorf("count all audit logs by user id query failed: %w", err)
		}
	} else if action != nil {
		logs, err = r.queries.FindAllAuditLogsByAction(ctx, sqlc.FindAllAuditLogsByActionParams{
			Action: *action,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all audit logs by action query failed: %w", err)
		}

		total, err = r.queries.CountAllAuditLogsByAction(ctx, *action)
		if err != nil {
			return nil, 0, fmt.Errorf("count all audit logs by action query failed: %w", err)
		}
	} else {
		logs, err = r.queries.FindAllAuditLogs(ctx, sqlc.FindAllAuditLogsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all audit logs query failed: %w", err)
		}

		total, err = r.queries.CountAllAuditLogs(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("count all audit logs query failed: %w", err)
		}
	}

	domainLogs := make([]model.AuditLog, len(logs))
	for i, l := range logs {
		domainLogs[i] = *mapAuditLog(l)
	}
	return domainLogs, int(total), nil
}

func mapAuditLog(l sqlc.AuditLog) *model.AuditLog {
	var userID uuid.UUID
	if l.UserID.Valid {
		userID = l.UserID.Bytes
	}
	var createdAt time.Time
	if l.CreatedAt.Valid {
		createdAt = l.CreatedAt.Time
	}

	return &model.AuditLog{
		ID:        l.ID,
		UserID:    userID,
		Action:    l.Action,
		IPAddress: l.IpAddress,
		Endpoint:  l.Endpoint,
		CreatedAt: createdAt,
	}
}
