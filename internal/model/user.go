package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserRole string

const (
	UserRoleUser  UserRole = "USER"
	UserRoleAdmin UserRole = "ADMIN"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never expose in JSON
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserWithWallet struct {
	User          User
	WalletID      string
	WalletBalance decimal.Decimal
}
