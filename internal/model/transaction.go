package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionTypeTopUp    TransactionType = "TOPUP"
	TransactionTypeTransfer TransactionType = "TRANSFER"

	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

type Transaction struct {
	ID             uuid.UUID         `json:"transaction_id"`
	ReferenceNo    string            `json:"reference_no"`
	Type           TransactionType   `json:"type"`
	Status         TransactionStatus `json:"status"`
	Amount         decimal.Decimal   `json:"amount"`
	SourceWalletID *string           `json:"source_wallet_id"` // nullable for TOPUP
	TargetWalletID string            `json:"target_wallet_id"`
	CreatedAt      time.Time         `json:"created_at"`
}
