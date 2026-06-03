package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/middleware"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type NotificationHandler struct {
	notifService service.NotificationService
	redisClient  *redis.Client
}

func NewNotificationHandler(notifService service.NotificationService, redisClient *redis.Client) *NotificationHandler {
	return &NotificationHandler{
		notifService: notifService,
		redisClient:  redisClient,
	}
}

func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	res, err := h.notifService.GetNotifications(c.Request.Context(), userID, pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Notifications retrieved successfully", res)
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Invalid notification ID format", nil)
		return
	}

	err = h.notifService.MarkAsRead(c.Request.Context(), id, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Notification marked as read successfully", nil)
}

func (h *NotificationHandler) ClearAll(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	err = h.notifService.ClearAll(c.Request.Context(), userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Notifications cleared successfully", nil)
}

func (h *NotificationHandler) StreamNotifications(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	channelName := fmt.Sprintf("user:notifications:%s", userID.String())
	pubsub := h.redisClient.Subscribe(c.Request.Context(), channelName)
	defer pubsub.Close()

	ch := pubsub.Channel()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("notification", msg.Payload)
			return true
		}
	})
}
