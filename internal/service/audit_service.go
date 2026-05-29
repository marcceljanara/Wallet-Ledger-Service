package service

import (
	"context"

	"github.com/google/uuid"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type AuditService interface {
	CreateLog(ctx context.Context, log *model.AuditLog) error
	GetLogs(ctx context.Context, userID uuid.UUID, pagination dto.PaginationRequest) (*dto.AuditLogListResponse, error)
	GetAllLogs(ctx context.Context, userID *uuid.UUID, action *string, pagination dto.PaginationRequest) (*dto.AuditLogListResponse, error)
}

type auditService struct {
	auditRepo repository.AuditRepository
}

func NewAuditService(auditRepo repository.AuditRepository) AuditService {
	return &auditService{
		auditRepo: auditRepo,
	}
}

func (s *auditService) CreateLog(ctx context.Context, log *model.AuditLog) error {
	id, err := utils.GenerateAuditLogID()
	if err != nil {
		return err
	}
	log.ID = id
	_, err = s.auditRepo.Create(ctx, log)
	return err
}

func (s *auditService) GetLogs(ctx context.Context, userID uuid.UUID, pagination dto.PaginationRequest) (*dto.AuditLogListResponse, error) {
	pagination.SetDefaults()

	logs, total, err := s.auditRepo.FindByUserID(ctx, userID, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	resLogs := make([]dto.AuditLogResponse, len(logs))
	for i, l := range logs {
		resLogs[i] = dto.AuditLogResponse{
			LogID:     l.ID,
			UserID:    l.UserID,
			Action:    l.Action,
			IPAddress: l.IPAddress,
			Endpoint:  l.Endpoint,
			CreatedAt: l.CreatedAt,
		}
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.AuditLogListResponse{
		Logs: resLogs,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func (s *auditService) GetAllLogs(ctx context.Context, userID *uuid.UUID, action *string, pagination dto.PaginationRequest) (*dto.AuditLogListResponse, error) {
	pagination.SetDefaults()

	logs, total, err := s.auditRepo.FindAll(ctx, userID, action, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	resLogs := make([]dto.AuditLogResponse, len(logs))
	for i, l := range logs {
		resLogs[i] = dto.AuditLogResponse{
			LogID:     l.ID,
			UserID:    l.UserID,
			Action:    l.Action,
			IPAddress: l.IPAddress,
			Endpoint:  l.Endpoint,
			CreatedAt: l.CreatedAt,
		}
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.AuditLogListResponse{
		Logs: resLogs,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}
