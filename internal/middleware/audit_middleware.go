package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type AuditEventMessage struct {
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	IPAddress string `json:"ip_address"`
	Endpoint  string `json:"endpoint"`
}

type WalletEvent struct {
	EventType string            `json:"event_type"`
	UserID    string            `json:"user_id"`
	Data      AuditEventMessage `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
	IPAddress string            `json:"ip_address"`
	Endpoint  string            `json:"endpoint"`
}

func AuditLog(rabbitCh *utils.SafeChannel) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		userIDVal, exists := c.Get("userID")
		if !exists {
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			return
		}

		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()

		action := getAction(method, path)

		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			evtMsg := AuditEventMessage{
				UserID:    userID.String(),
				Action:    action,
				IPAddress: ip,
				Endpoint:  method + " " + path,
			}

			evt := WalletEvent{
				EventType: "AUDIT",
				UserID:    userID.String(),
				Data:      evtMsg,
				Timestamp: time.Now(),
				IPAddress: ip,
				Endpoint:  method + " " + path,
			}

			body, err := json.Marshal(evt)
			if err != nil {
				slog.Error("failed to marshal audit event", "error", err)
				return
			}

			if rabbitCh != nil {
				err = rabbitCh.PublishWithContext(bgCtx,
					"wallet_events",
					"wallet.event.audit",
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				)
				if err != nil {
					slog.Error("failed to publish audit event to rabbitmq", "error", err)
				}
			}
		}()
	}
}

func getAction(method, path string) string {
	switch {
	case method == "POST" && strings.HasSuffix(path, "/wallets/topup"):
		return "TOPUP"
	case method == "POST" && strings.HasSuffix(path, "/transfers"):
		return "TRANSFER"
	case method == "POST" && strings.HasSuffix(path, "/auth/register"):
		return "REGISTER"
	case method == "POST" && strings.HasSuffix(path, "/auth/login"):
		return "LOGIN"
	case method == "POST" && strings.HasSuffix(path, "/auth/logout"):
		return "LOGOUT"
	case method == "GET" && strings.HasSuffix(path, "/wallets/me"):
		return "VIEW_BALANCE"
	case method == "GET" && strings.HasSuffix(path, "/admin/transactions"):
		return "ADMIN_VIEW_TRANSACTIONS"
	case method == "GET" && (strings.Contains(path, "/transactions/") || strings.Contains(path, "/admin/transactions/")):
		if strings.Contains(path, "/admin/") {
			return "ADMIN_VIEW_TRANSACTION_DETAIL"
		}
		return "VIEW_TRANSACTION_DETAIL"
	case method == "GET" && strings.HasSuffix(path, "/transactions"):
		return "VIEW_TRANSACTIONS"
	case method == "GET" && strings.HasSuffix(path, "/ledger/entries"):
		return "VIEW_LEDGER"
	case method == "GET" && strings.HasSuffix(path, "/audit/logs"):
		return "VIEW_AUDIT_LOGS"
	case method == "GET" && strings.HasSuffix(path, "/admin/audit-logs"):
		return "ADMIN_VIEW_AUDIT_LOGS"
	case method == "GET" && strings.HasSuffix(path, "/admin/users"):
		return "ADMIN_VIEW_USERS"
	default:
		return method + "_" + path
	}
}
