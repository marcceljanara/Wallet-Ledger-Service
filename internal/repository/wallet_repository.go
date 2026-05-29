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
	"github.com/shopspring/decimal"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository/sqlc"
)

type WalletRepository interface {
	Create(ctx context.Context, tx pgx.Tx, wallet *model.Wallet) (*model.Wallet, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error)
	FindByID(ctx context.Context, walletID string) (*model.Wallet, error)
	FindByIDForUpdate(ctx context.Context, tx pgx.Tx, walletID string) (*model.Wallet, error)
	UpdateBalance(ctx context.Context, tx pgx.Tx, walletID string, newBalance decimal.Decimal) error
}

type walletRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewWalletRepository(pool *pgxpool.Pool) WalletRepository {
	return &walletRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *walletRepository) getQueries(tx pgx.Tx) *sqlc.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *walletRepository) Create(ctx context.Context, tx pgx.Tx, wallet *model.Wallet) (*model.Wallet, error) {
	q := r.getQueries(tx)
	pgUUID := pgtype.UUID{Bytes: wallet.UserID, Valid: true}
	pgBalance := decimalToNumeric(wallet.Balance)

	w, err := q.CreateWallet(ctx, sqlc.CreateWalletParams{
		ID:       wallet.ID,
		UserID:   pgUUID,
		Balance:  pgBalance,
		Currency: wallet.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("create wallet query failed: %w", err)
	}
	return mapWallet(w), nil
}

func (r *walletRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	pgUUID := pgtype.UUID{Bytes: userID, Valid: true}
	w, err := r.queries.FindWalletByUserID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find wallet by user id query failed: %w", err)
	}
	return mapWallet(w), nil
}

func (r *walletRepository) FindByID(ctx context.Context, walletID string) (*model.Wallet, error) {
	w, err := r.queries.FindWalletByID(ctx, walletID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find wallet by id query failed: %w", err)
	}
	return mapWallet(w), nil
}

func (r *walletRepository) FindByIDForUpdate(ctx context.Context, tx pgx.Tx, walletID string) (*model.Wallet, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction is required for find wallet by id for update")
	}
	q := r.queries.WithTx(tx)
	w, err := q.FindWalletByIDForUpdate(ctx, walletID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find wallet by id for update query failed: %w", err)
	}
	return mapWallet(w), nil
}

func (r *walletRepository) UpdateBalance(ctx context.Context, tx pgx.Tx, walletID string, newBalance decimal.Decimal) error {
	q := r.getQueries(tx)
	pgBalance := decimalToNumeric(newBalance)
	err := q.UpdateWalletBalance(ctx, sqlc.UpdateWalletBalanceParams{
		ID:      walletID,
		Balance: pgBalance,
	})
	if err != nil {
		return fmt.Errorf("update wallet balance query failed: %w", err)
	}
	return nil
}

func mapWallet(w sqlc.Wallet) *model.Wallet {
	var userID uuid.UUID
	if w.UserID.Valid {
		userID = w.UserID.Bytes
	}
	var createdAt time.Time
	if w.CreatedAt.Valid {
		createdAt = w.CreatedAt.Time
	}
	var updatedAt time.Time
	if w.UpdatedAt.Valid {
		updatedAt = w.UpdatedAt.Time
	}
	return &model.Wallet{
		ID:        w.ID,
		UserID:    userID,
		Balance:   numericToDecimal(w.Balance),
		Currency:  w.Currency,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func decimalToNumeric(d decimal.Decimal) pgtype.Numeric {
	var num pgtype.Numeric
	err := num.Scan(d.String())
	if err != nil {
		num.Valid = false
	}
	return num
}

func numericToDecimal(n pgtype.Numeric) decimal.Decimal {
	if !n.Valid {
		return decimal.Zero
	}
	if n.Int == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(n.Int, n.Exp)
}
