package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionResponse struct {
	TransactionID  uuid.UUID       `json:"transaction_id"`
	ReferenceNo    string          `json:"reference_no"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	Amount         decimal.Decimal `json:"amount"`
	SourceWalletID *string         `json:"source_wallet_id"`
	TargetWalletID string          `json:"target_wallet_id"`
	CreatedAt      time.Time       `json:"created_at"`
}

type TransactionDetailResponse struct {
	TransactionResponse
	LedgerEntries []LedgerEntryResponse `json:"ledger_entries"`
}

type TransactionListResponse struct {
	Transactions []TransactionResponse `json:"transactions"`
	Pagination   PaginationResponse    `json:"pagination"`
}
