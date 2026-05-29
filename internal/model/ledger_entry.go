package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type EntryType string

const (
	EntryTypeDebit  EntryType = "DEBIT"
	EntryTypeCredit EntryType = "CREDIT"
)

type LedgerEntry struct {
	ID            string          `json:"entry_id"`
	TransactionID uuid.UUID       `json:"transaction_id"`
	WalletID      string          `json:"wallet_id"`
	EntryType     EntryType       `json:"entry_type"`
	Amount        decimal.Decimal `json:"amount"`
	CreatedAt     time.Time       `json:"created_at"`
}
