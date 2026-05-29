package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletResponse struct {
	WalletID  string          `json:"wallet_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Balance   decimal.Decimal `json:"balance"`
	Currency  string          `json:"currency"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
