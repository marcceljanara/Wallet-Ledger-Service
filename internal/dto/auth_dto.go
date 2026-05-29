package dto

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	WalletID  string    `json:"wallet_id"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginResponse struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
}
