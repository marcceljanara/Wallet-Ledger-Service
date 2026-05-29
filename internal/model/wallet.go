package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID        string          `json:"wallet_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Balance   decimal.Decimal `json:"balance"`
	Currency  string          `json:"currency"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
