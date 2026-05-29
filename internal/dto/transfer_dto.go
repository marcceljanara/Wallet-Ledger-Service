package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransferRequest struct {
	TargetWalletID string          `json:"target_wallet_id" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required,gt=0"`
}

type TransferResponse struct {
	TransactionID  uuid.UUID       `json:"transaction_id"`
	ReferenceNo    string          `json:"reference_no"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	Amount         decimal.Decimal `json:"amount"`
	SourceWalletID string          `json:"source_wallet_id"`
	TargetWalletID string          `json:"target_wallet_id"`
	BalanceAfter   decimal.Decimal `json:"balance_after"`
	CreatedAt      time.Time       `json:"created_at"`
}
