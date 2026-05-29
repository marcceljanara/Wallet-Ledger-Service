package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository/sqlc"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx pgx.Tx, txn *model.Transaction) (*model.Transaction, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
	UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status model.TransactionStatus) error
	FindByWalletID(ctx context.Context, walletID string, txnType *string, limit, offset int) ([]model.Transaction, int, error)
	FindAll(ctx context.Context, txnType, txnStatus *string, limit, offset int) ([]model.Transaction, int, error)
}

type transactionRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewTransactionRepository(pool *pgxpool.Pool) TransactionRepository {
	return &transactionRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *transactionRepository) getQueries(tx pgx.Tx) *sqlc.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *transactionRepository) Create(ctx context.Context, tx pgx.Tx, txn *model.Transaction) (*model.Transaction, error) {
	q := r.getQueries(tx)
	pgAmount := decimalToNumeric(txn.Amount)

	var sourceWalletID pgtype.Text
	if txn.SourceWalletID != nil {
		sourceWalletID = pgtype.Text{String: *txn.SourceWalletID, Valid: true}
	}

	var targetWalletID pgtype.Text
	if txn.TargetWalletID != "" {
		targetWalletID = pgtype.Text{String: txn.TargetWalletID, Valid: true}
	}

	t, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		ReferenceNo:    txn.ReferenceNo,
		Type:           string(txn.Type),
		Status:         string(txn.Status),
		Amount:         pgAmount,
		SourceWalletID: sourceWalletID,
		TargetWalletID: targetWalletID,
	})
	if err != nil {
		return nil, fmt.Errorf("create transaction query failed: %w", err)
	}
	return mapTransaction(t), nil
}

func (r *transactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	t, err := r.queries.FindTransactionByID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find transaction by id query failed: %w", err)
	}
	return mapTransaction(t), nil
}

func (r *transactionRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status model.TransactionStatus) error {
	q := r.getQueries(tx)
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	err := q.UpdateTransactionStatus(ctx, sqlc.UpdateTransactionStatusParams{
		ID:     pgUUID,
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("update transaction status query failed: %w", err)
	}
	return nil
}

func (r *transactionRepository) FindByWalletID(ctx context.Context, walletID string, txnType *string, limit, offset int) ([]model.Transaction, int, error) {
	var txns []sqlc.Transaction
	var err error
	var total int64

	pgSource := pgtype.Text{String: walletID, Valid: true}

	if txnType != nil {
		txns, err = r.queries.FindTransactionsByWalletIDAndType(ctx, sqlc.FindTransactionsByWalletIDAndTypeParams{
			SourceWalletID: pgSource,
			Type:           *txnType,
			Limit:          int32(limit),
			Offset:         int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find transactions by wallet id and type query failed: %w", err)
		}

		total, err = r.queries.CountTransactionsByWalletIDAndType(ctx, sqlc.CountTransactionsByWalletIDAndTypeParams{
			SourceWalletID: pgSource,
			Type:           *txnType,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("count transactions by wallet id and type query failed: %w", err)
		}
	} else {
		txns, err = r.queries.FindTransactionsByWalletID(ctx, sqlc.FindTransactionsByWalletIDParams{
			SourceWalletID: pgSource,
			Limit:          int32(limit),
			Offset:         int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find transactions by wallet id query failed: %w", err)
		}

		total, err = r.queries.CountTransactionsByWalletID(ctx, pgSource)
		if err != nil {
			return nil, 0, fmt.Errorf("count transactions by wallet id query failed: %w", err)
		}
	}

	domainTxns := make([]model.Transaction, len(txns))
	for i, t := range txns {
		domainTxns[i] = *mapTransaction(t)
	}

	return domainTxns, int(total), nil
}

func (r *transactionRepository) FindAll(ctx context.Context, txnType, txnStatus *string, limit, offset int) ([]model.Transaction, int, error) {
	var txns []sqlc.Transaction
	var err error
	var total int64

	if txnType != nil && txnStatus != nil {
		txns, err = r.queries.FindAllTransactionsByTypeAndStatus(ctx, sqlc.FindAllTransactionsByTypeAndStatusParams{
			Type:   *txnType,
			Status: *txnStatus,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all transactions by type and status failed: %w", err)
		}
		total, err = r.queries.CountAllTransactionsByTypeAndStatus(ctx, sqlc.CountAllTransactionsByTypeAndStatusParams{
			Type:   *txnType,
			Status: *txnStatus,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("count all transactions by type and status failed: %w", err)
		}
	} else if txnType != nil {
		txns, err = r.queries.FindAllTransactionsByType(ctx, sqlc.FindAllTransactionsByTypeParams{
			Type:   *txnType,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all transactions by type failed: %w", err)
		}
		total, err = r.queries.CountAllTransactionsByType(ctx, *txnType)
		if err != nil {
			return nil, 0, fmt.Errorf("count all transactions by type failed: %w", err)
		}
	} else if txnStatus != nil {
		txns, err = r.queries.FindAllTransactionsByStatus(ctx, sqlc.FindAllTransactionsByStatusParams{
			Status: *txnStatus,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all transactions by status failed: %w", err)
		}
		total, err = r.queries.CountAllTransactionsByStatus(ctx, *txnStatus)
		if err != nil {
			return nil, 0, fmt.Errorf("count all transactions by status failed: %w", err)
		}
	} else {
		txns, err = r.queries.FindAllTransactions(ctx, sqlc.FindAllTransactionsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find all transactions failed: %w", err)
		}
		total, err = r.queries.CountAllTransactions(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("count all transactions failed: %w", err)
		}
	}

	domainTxns := make([]model.Transaction, len(txns))
	for i, t := range txns {
		domainTxns[i] = *mapTransaction(t)
	}

	return domainTxns, int(total), nil
}

func mapTransaction(t sqlc.Transaction) *model.Transaction {
	var id uuid.UUID
	if t.ID.Valid {
		id = t.ID.Bytes
	}
	var srcWallet *string
	if t.SourceWalletID.Valid {
		srcWallet = &t.SourceWalletID.String
	}
	var tgtWallet string
	if t.TargetWalletID.Valid {
		tgtWallet = t.TargetWalletID.String
	}
	var createdAt time.Time
	if t.CreatedAt.Valid {
		createdAt = t.CreatedAt.Time
	}

	return &model.Transaction{
		ID:             id,
		ReferenceNo:    t.ReferenceNo,
		Type:           model.TransactionType(t.Type),
		Status:         model.TransactionStatus(t.Status),
		Amount:         numericToDecimal(t.Amount),
		SourceWalletID: srcWallet,
		TargetWalletID: tgtWallet,
		CreatedAt:      createdAt,
	}
}
