package service

import (
	"context"

	"github.com/google/uuid"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/repository"
)

type LedgerService interface {
	GetLedgerEntries(ctx context.Context, userID uuid.UUID, entryType *string, pagination dto.PaginationRequest) (*dto.LedgerListResponse, error)
}

type ledgerService struct {
	walletRepo repository.WalletRepository
	ledgerRepo repository.LedgerRepository
}

func NewLedgerService(walletRepo repository.WalletRepository, ledgerRepo repository.LedgerRepository) LedgerService {
	return &ledgerService{
		walletRepo: walletRepo,
		ledgerRepo: ledgerRepo,
	}
}

func (s *ledgerService) GetLedgerEntries(ctx context.Context, userID uuid.UUID, entryType *string, pagination dto.PaginationRequest) (*dto.LedgerListResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	pagination.SetDefaults()

	entries, total, err := s.ledgerRepo.FindByWalletID(ctx, wallet.ID, entryType, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	resEntries := make([]dto.LedgerEntryResponse, len(entries))
	for i, e := range entries {
		resEntries[i] = dto.LedgerEntryResponse{
			EntryID:       e.ID,
			TransactionID: e.TransactionID,
			WalletID:      e.WalletID,
			EntryType:     string(e.EntryType),
			Amount:        e.Amount,
			CreatedAt:     e.CreatedAt,
		}
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.LedgerListResponse{
		Entries: resEntries,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}
