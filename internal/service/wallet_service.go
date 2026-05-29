package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/repository"
)

var ErrWalletNotFound = errors.New("wallet not found")

type WalletService interface {
	GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*dto.WalletResponse, error)
}

type walletService struct {
	walletRepo repository.WalletRepository
}

func NewWalletService(walletRepo repository.WalletRepository) WalletService {
	return &walletService{
		walletRepo: walletRepo,
	}
}

func (s *walletService) GetWalletByUserID(ctx context.Context, userID uuid.UUID) (*dto.WalletResponse, error) {
	w, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWalletNotFound
	}

	return &dto.WalletResponse{
		WalletID:  w.ID,
		UserID:    w.UserID,
		Balance:   w.Balance,
		Currency:  w.Currency,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}, nil
}
