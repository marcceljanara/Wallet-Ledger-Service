package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

var (
	ErrSelfTransfer         = errors.New("cannot transfer to your own wallet")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrUnauthorized         = errors.New("unauthorized to view this transaction")
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrTargetWalletNotFound = errors.New("target wallet not found")
)

type TransactionService interface {
	TopUp(ctx context.Context, userID uuid.UUID, req dto.TopUpRequest) (*dto.TopUpResponse, error)
	Transfer(ctx context.Context, userID uuid.UUID, req dto.TransferRequest) (*dto.TransferResponse, error)
	GetTransactions(ctx context.Context, userID uuid.UUID, txnType *string, pagination dto.PaginationRequest) (*dto.TransactionListResponse, error)
	GetTransactionDetail(ctx context.Context, userID uuid.UUID, transactionID uuid.UUID) (*dto.TransactionDetailResponse, error)
	GetAllTransactions(ctx context.Context, txnType, txnStatus *string, pagination dto.PaginationRequest) (*dto.TransactionListResponse, error)
	GetTransactionDetailAdmin(ctx context.Context, transactionID uuid.UUID) (*dto.TransactionDetailResponse, error)
}

type transactionService struct {
	walletRepo      repository.WalletRepository
	transactionRepo repository.TransactionRepository
	ledgerRepo      repository.LedgerRepository
	pool            TxBeginner
	rabbitCh        *amqp.Channel
}

func NewTransactionService(
	walletRepo repository.WalletRepository,
	transactionRepo repository.TransactionRepository,
	ledgerRepo repository.LedgerRepository,
	pool TxBeginner,
	rabbitCh *amqp.Channel,
) TransactionService {
	return &transactionService{
		walletRepo:      walletRepo,
		transactionRepo: transactionRepo,
		ledgerRepo:      ledgerRepo,
		pool:            pool,
		rabbitCh:        rabbitCh,
	}
}

func (s *transactionService) TopUp(ctx context.Context, userID uuid.UUID, req dto.TopUpRequest) (*dto.TopUpResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	lockedWallet, err := s.walletRepo.FindByIDForUpdate(ctx, tx, wallet.ID)
	if err != nil {
		return nil, err
	}
	if lockedWallet == nil {
		return nil, ErrWalletNotFound
	}

	txRef := utils.GenerateTransactionRef()
	txID := uuid.New()

	txn := &model.Transaction{
		ID:             txID,
		ReferenceNo:    txRef,
		Type:           model.TransactionTypeTopUp,
		Status:         model.TransactionStatusPending,
		Amount:         req.Amount,
		SourceWalletID: nil,
		TargetWalletID: wallet.ID,
		CreatedAt:      time.Now(),
	}

	txn, err = s.transactionRepo.Create(ctx, tx, txn)
	if err != nil {
		return nil, err
	}

	newBalance := lockedWallet.Balance.Add(req.Amount)
	err = s.walletRepo.UpdateBalance(ctx, tx, wallet.ID, newBalance)
	if err != nil {
		return nil, err
	}

	ledgerID := utils.GenerateLedgerEntryID()
	ledgerEntry := &model.LedgerEntry{
		ID:            ledgerID,
		TransactionID: txn.ID,
		WalletID:      wallet.ID,
		EntryType:     model.EntryTypeCredit,
		Amount:        req.Amount,
		CreatedAt:     time.Now(),
	}

	_, err = s.ledgerRepo.Create(ctx, tx, ledgerEntry)
	if err != nil {
		return nil, err
	}

	err = s.transactionRepo.UpdateStatus(ctx, tx, txn.ID, model.TransactionStatusCompleted)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		eventData := map[string]interface{}{
			"transaction_id": txn.ID.String(),
			"reference_no":   txn.ReferenceNo,
			"wallet_id":      wallet.ID,
			"amount":         req.Amount,
			"balance_after":  newBalance,
		}
		if err := publishEvent(bgCtx, s.rabbitCh, "TOPUP", userID.String(), "", "", eventData); err != nil {
			slog.Error("failed to publish topup event to rabbitmq", "error", err)
		}
	}()

	return &dto.TopUpResponse{
		TransactionID: txn.ID,
		ReferenceNo:   txn.ReferenceNo,
		Type:          string(txn.Type),
		Status:        string(model.TransactionStatusCompleted),
		Amount:        req.Amount,
		WalletID:      wallet.ID,
		BalanceAfter:  newBalance,
		CreatedAt:     txn.CreatedAt,
	}, nil
}

func (s *transactionService) Transfer(ctx context.Context, userID uuid.UUID, req dto.TransferRequest) (*dto.TransferResponse, error) {
	sourceWallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sourceWallet == nil {
		return nil, ErrWalletNotFound
	}

	if sourceWallet.ID == req.TargetWalletID {
		return nil, ErrSelfTransfer
	}

	targetWallet, err := s.walletRepo.FindByID(ctx, req.TargetWalletID)
	if err != nil {
		return nil, err
	}
	if targetWallet == nil {
		return nil, ErrTargetWalletNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var w1, w2 *model.Wallet
	if sourceWallet.ID < targetWallet.ID {
		w1, err = s.walletRepo.FindByIDForUpdate(ctx, tx, sourceWallet.ID)
		if err != nil {
			return nil, err
		}
		w2, err = s.walletRepo.FindByIDForUpdate(ctx, tx, targetWallet.ID)
		if err != nil {
			return nil, err
		}
	} else {
		w2, err = s.walletRepo.FindByIDForUpdate(ctx, tx, targetWallet.ID)
		if err != nil {
			return nil, err
		}
		w1, err = s.walletRepo.FindByIDForUpdate(ctx, tx, sourceWallet.ID)
		if err != nil {
			return nil, err
		}
	}

	var lockedSource, lockedTarget *model.Wallet
	if w1.ID == sourceWallet.ID {
		lockedSource = w1
		lockedTarget = w2
	} else {
		lockedSource = w2
		lockedTarget = w1
	}

	if lockedSource.Balance.LessThan(req.Amount) {
		return nil, ErrInsufficientBalance
	}

	txRef := utils.GenerateTransactionRef()
	txID := uuid.New()

	txn := &model.Transaction{
		ID:             txID,
		ReferenceNo:    txRef,
		Type:           model.TransactionTypeTransfer,
		Status:         model.TransactionStatusPending,
		Amount:         req.Amount,
		SourceWalletID: &lockedSource.ID,
		TargetWalletID: lockedTarget.ID,
		CreatedAt:      time.Now(),
	}

	txn, err = s.transactionRepo.Create(ctx, tx, txn)
	if err != nil {
		return nil, err
	}

	sourceNewBalance := lockedSource.Balance.Sub(req.Amount)
	err = s.walletRepo.UpdateBalance(ctx, tx, lockedSource.ID, sourceNewBalance)
	if err != nil {
		return nil, err
	}

	targetNewBalance := lockedTarget.Balance.Add(req.Amount)
	err = s.walletRepo.UpdateBalance(ctx, tx, lockedTarget.ID, targetNewBalance)
	if err != nil {
		return nil, err
	}

	debitID := utils.GenerateLedgerEntryID()
	debitEntry := &model.LedgerEntry{
		ID:            debitID,
		TransactionID: txn.ID,
		WalletID:      lockedSource.ID,
		EntryType:     model.EntryTypeDebit,
		Amount:        req.Amount,
		CreatedAt:     time.Now(),
	}
	_, err = s.ledgerRepo.Create(ctx, tx, debitEntry)
	if err != nil {
		return nil, err
	}

	creditID := utils.GenerateLedgerEntryID()
	creditEntry := &model.LedgerEntry{
		ID:            creditID,
		TransactionID: txn.ID,
		WalletID:      lockedTarget.ID,
		EntryType:     model.EntryTypeCredit,
		Amount:        req.Amount,
		CreatedAt:     time.Now(),
	}
	_, err = s.ledgerRepo.Create(ctx, tx, creditEntry)
	if err != nil {
		return nil, err
	}

	err = s.transactionRepo.UpdateStatus(ctx, tx, txn.ID, model.TransactionStatusCompleted)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		eventData := map[string]interface{}{
			"transaction_id":   txn.ID.String(),
			"reference_no":     txn.ReferenceNo,
			"source_wallet_id": lockedSource.ID,
			"target_wallet_id": lockedTarget.ID,
			"amount":           req.Amount,
			"balance_after":    sourceNewBalance,
		}
		if err := publishEvent(bgCtx, s.rabbitCh, "TRANSFER", userID.String(), "", "", eventData); err != nil {
			slog.Error("failed to publish transfer event to rabbitmq", "error", err)
		}
	}()

	return &dto.TransferResponse{
		TransactionID:  txn.ID,
		ReferenceNo:    txn.ReferenceNo,
		Type:           string(txn.Type),
		Status:         string(model.TransactionStatusCompleted),
		Amount:         req.Amount,
		SourceWalletID: lockedSource.ID,
		TargetWalletID: lockedTarget.ID,
		BalanceAfter:   sourceNewBalance,
		CreatedAt:      txn.CreatedAt,
	}, nil
}

func (s *transactionService) GetTransactions(ctx context.Context, userID uuid.UUID, txnType *string, pagination dto.PaginationRequest) (*dto.TransactionListResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	pagination.SetDefaults()

	txns, total, err := s.transactionRepo.FindByWalletID(ctx, wallet.ID, txnType, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	resTxns := make([]dto.TransactionResponse, len(txns))
	for i, t := range txns {
		resTxns[i] = dto.TransactionResponse{
			TransactionID:  t.ID,
			ReferenceNo:    t.ReferenceNo,
			Type:           string(t.Type),
			Status:         string(t.Status),
			Amount:         t.Amount,
			SourceWalletID: t.SourceWalletID,
			TargetWalletID: t.TargetWalletID,
			CreatedAt:      t.CreatedAt,
		}
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.TransactionListResponse{
		Transactions: resTxns,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func (s *transactionService) GetTransactionDetail(ctx context.Context, userID uuid.UUID, transactionID uuid.UUID) (*dto.TransactionDetailResponse, error) {
	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	txn, err := s.transactionRepo.FindByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if txn == nil {
		return nil, ErrTransactionNotFound
	}

	isAuthorized := false
	if txn.TargetWalletID == wallet.ID {
		isAuthorized = true
	} else if txn.SourceWalletID != nil && *txn.SourceWalletID == wallet.ID {
		isAuthorized = true
	}

	if !isAuthorized {
		return nil, ErrUnauthorized
	}

	entries, err := s.ledgerRepo.FindByTransactionID(ctx, txn.ID)
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

	return &dto.TransactionDetailResponse{
		TransactionResponse: dto.TransactionResponse{
			TransactionID:  txn.ID,
			ReferenceNo:    txn.ReferenceNo,
			Type:           string(txn.Type),
			Status:         string(txn.Status),
			Amount:         txn.Amount,
			SourceWalletID: txn.SourceWalletID,
			TargetWalletID: txn.TargetWalletID,
			CreatedAt:      txn.CreatedAt,
		},
		LedgerEntries: resEntries,
	}, nil
}

func (s *transactionService) GetAllTransactions(ctx context.Context, txnType, txnStatus *string, pagination dto.PaginationRequest) (*dto.TransactionListResponse, error) {
	pagination.SetDefaults()

	txns, total, err := s.transactionRepo.FindAll(ctx, txnType, txnStatus, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	resTxns := make([]dto.TransactionResponse, len(txns))
	for i, t := range txns {
		resTxns[i] = dto.TransactionResponse{
			TransactionID:  t.ID,
			ReferenceNo:    t.ReferenceNo,
			Type:           string(t.Type),
			Status:         string(t.Status),
			Amount:         t.Amount,
			SourceWalletID: t.SourceWalletID,
			TargetWalletID: t.TargetWalletID,
			CreatedAt:      t.CreatedAt,
		}
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.TransactionListResponse{
		Transactions: resTxns,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func (s *transactionService) GetTransactionDetailAdmin(ctx context.Context, transactionID uuid.UUID) (*dto.TransactionDetailResponse, error) {
	txn, err := s.transactionRepo.FindByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if txn == nil {
		return nil, ErrTransactionNotFound
	}

	entries, err := s.ledgerRepo.FindByTransactionID(ctx, txn.ID)
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

	return &dto.TransactionDetailResponse{
		TransactionResponse: dto.TransactionResponse{
			TransactionID:  txn.ID,
			ReferenceNo:    txn.ReferenceNo,
			Type:           string(txn.Type),
			Status:         string(txn.Status),
			Amount:         txn.Amount,
			SourceWalletID: txn.SourceWalletID,
			TargetWalletID: txn.TargetWalletID,
			CreatedAt:      txn.CreatedAt,
		},
		LedgerEntries: resEntries,
	}, nil
}
