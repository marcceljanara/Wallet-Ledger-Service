package dto

import (
	"time"

	"github.com/google/uuid"
)

type AuditLogResponse struct {
	LogID     string    `json:"log_id"`
	UserID    uuid.UUID `json:"user_id"`
	Action    string    `json:"action"`
	IPAddress string    `json:"ip_address"`
	Endpoint  string    `json:"endpoint"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditLogListResponse struct {
	Logs       []AuditLogResponse `json:"logs"`
	Pagination PaginationResponse `json:"pagination"`
}
