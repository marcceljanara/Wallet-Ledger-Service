package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/middleware"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type AuditHandler struct {
	auditService service.AuditService
}

func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
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

	res, err := h.auditService.GetLogs(c.Request.Context(), userID, pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Audit logs retrieved successfully", res)
}

func (h *AuditHandler) GetAuditLogsAdmin(c *gin.Context) {
	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	var userID *uuid.UUID
	if uidStr := c.Query("user_id"); uidStr != "" {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			utils.WriteError(c, http.StatusBadRequest, "Invalid user_id format", nil)
			return
		}
		userID = &uid
	}

	var action *string
	if act := c.Query("action"); act != "" {
		action = &act
	}

	res, err := h.auditService.GetAllLogs(c.Request.Context(), userID, action, pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Audit logs retrieved successfully", res)
}
