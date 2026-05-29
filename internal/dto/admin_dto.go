package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AdminUserResponse struct {
	UserID    uuid.UUID       `json:"user_id"`
	Email     string          `json:"email"`
	Role      string          `json:"role"`
	WalletID  string          `json:"wallet_id"`
	Balance   decimal.Decimal `json:"balance"`
	CreatedAt time.Time       `json:"created_at"`
}

type AdminUserListResponse struct {
	Users      []AdminUserResponse `json:"users"`
	Pagination PaginationResponse  `json:"pagination"`
}
