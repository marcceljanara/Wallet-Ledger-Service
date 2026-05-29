package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type LedgerEntryResponse struct {
	EntryID       string          `json:"entry_id"`
	TransactionID uuid.UUID       `json:"transaction_id"`
	WalletID      string          `json:"wallet_id"`
	EntryType     string          `json:"entry_type"`
	Amount        decimal.Decimal `json:"amount"`
	CreatedAt     time.Time       `json:"created_at"`
}

type LedgerListResponse struct {
	Entries    []LedgerEntryResponse `json:"entries"`
	Pagination PaginationResponse    `json:"pagination"`
}
