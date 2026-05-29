package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository/sqlc"
)

type LedgerRepository interface {
	Create(ctx context.Context, tx pgx.Tx, entry *model.LedgerEntry) (*model.LedgerEntry, error)
	FindByTransactionID(ctx context.Context, transactionID uuid.UUID) ([]model.LedgerEntry, error)
	FindByWalletID(ctx context.Context, walletID string, entryType *string, limit, offset int) ([]model.LedgerEntry, int, error)
}

type ledgerRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewLedgerRepository(pool *pgxpool.Pool) LedgerRepository {
	return &ledgerRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *ledgerRepository) getQueries(tx pgx.Tx) *sqlc.Queries {
	if tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

func (r *ledgerRepository) Create(ctx context.Context, tx pgx.Tx, entry *model.LedgerEntry) (*model.LedgerEntry, error) {
	q := r.getQueries(tx)
	pgUUID := pgtype.UUID{Bytes: entry.TransactionID, Valid: true}
	pgAmount := decimalToNumeric(entry.Amount)

	le, err := q.CreateLedgerEntry(ctx, sqlc.CreateLedgerEntryParams{
		ID:            entry.ID,
		TransactionID: pgUUID,
		WalletID:      entry.WalletID,
		EntryType:     string(entry.EntryType),
		Amount:        pgAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("create ledger entry query failed: %w", err)
	}
	return mapLedgerEntry(le), nil
}

func (r *ledgerRepository) FindByTransactionID(ctx context.Context, transactionID uuid.UUID) ([]model.LedgerEntry, error) {
	pgUUID := pgtype.UUID{Bytes: transactionID, Valid: true}
	entries, err := r.queries.FindLedgerEntriesByTransactionID(ctx, pgUUID)
	if err != nil {
		return nil, fmt.Errorf("find ledger entries by transaction id query failed: %w", err)
	}

	domainEntries := make([]model.LedgerEntry, len(entries))
	for i, le := range entries {
		domainEntries[i] = *mapLedgerEntry(le)
	}
	return domainEntries, nil
}

func (r *ledgerRepository) FindByWalletID(ctx context.Context, walletID string, entryType *string, limit, offset int) ([]model.LedgerEntry, int, error) {
	var entries []sqlc.LedgerEntry
	var err error
	var total int64

	if entryType != nil {
		entries, err = r.queries.FindLedgerEntriesByWalletIDAndType(ctx, sqlc.FindLedgerEntriesByWalletIDAndTypeParams{
			WalletID:  walletID,
			EntryType: *entryType,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find ledger entries by wallet id and type query failed: %w", err)
		}

		total, err = r.queries.CountLedgerEntriesByWalletIDAndType(ctx, sqlc.CountLedgerEntriesByWalletIDAndTypeParams{
			WalletID:  walletID,
			EntryType: *entryType,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("count ledger entries by wallet id and type query failed: %w", err)
		}
	} else {
		entries, err = r.queries.FindLedgerEntriesByWalletID(ctx, sqlc.FindLedgerEntriesByWalletIDParams{
			WalletID: walletID,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
		if err != nil {
			return nil, 0, fmt.Errorf("find ledger entries by wallet id query failed: %w", err)
		}

		total, err = r.queries.CountLedgerEntriesByWalletID(ctx, walletID)
		if err != nil {
			return nil, 0, fmt.Errorf("count ledger entries by wallet id query failed: %w", err)
		}
	}

	domainEntries := make([]model.LedgerEntry, len(entries))
	for i, le := range entries {
		domainEntries[i] = *mapLedgerEntry(le)
	}
	return domainEntries, int(total), nil
}

func mapLedgerEntry(le sqlc.LedgerEntry) *model.LedgerEntry {
	var txID uuid.UUID
	if le.TransactionID.Valid {
		txID = le.TransactionID.Bytes
	}
	var createdAt time.Time
	if le.CreatedAt.Valid {
		createdAt = le.CreatedAt.Time
	}

	return &model.LedgerEntry{
		ID:            le.ID,
		TransactionID: txID,
		WalletID:      le.WalletID,
		EntryType:     model.EntryType(le.EntryType),
		Amount:        numericToDecimal(le.Amount),
		CreatedAt:     createdAt,
	}
}
