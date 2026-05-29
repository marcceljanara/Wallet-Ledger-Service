package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/middleware"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type LedgerHandler struct {
	ledgerService service.LedgerService
}

func NewLedgerHandler(ledgerService service.LedgerService) *LedgerHandler {
	return &LedgerHandler{
		ledgerService: ledgerService,
	}
}

func (h *LedgerHandler) GetLedgerEntries(c *gin.Context) {
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

	var entryType *string
	if et := c.Query("entry_type"); et != "" {
		entryType = &et
	}

	res, err := h.ledgerService.GetLedgerEntries(c.Request.Context(), userID, entryType, pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Ledger entries retrieved successfully", res)
}
