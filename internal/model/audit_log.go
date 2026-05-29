package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        string    `json:"log_id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"`
	IPAddress string    `json:"ip_address"`
	Endpoint  string    `json:"endpoint"`
	CreatedAt time.Time `json:"created_at"`
}
