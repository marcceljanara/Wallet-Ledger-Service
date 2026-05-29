package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TopUpRequest struct {
	Amount decimal.Decimal `json:"amount" binding:"required,gt=0"`
}

type TopUpResponse struct {
	TransactionID uuid.UUID       `json:"transaction_id"`
	ReferenceNo   string          `json:"reference_no"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	Amount        decimal.Decimal `json:"amount"`
	WalletID      string          `json:"wallet_id"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	CreatedAt     time.Time       `json:"created_at"`
}
