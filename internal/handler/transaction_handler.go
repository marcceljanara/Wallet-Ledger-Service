package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/middleware"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type TransactionHandler struct {
	transactionService service.TransactionService
}

func NewTransactionHandler(transactionService service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

func (h *TransactionHandler) TopUp(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	var req dto.TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	res, err := h.transactionService.TopUp(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrWalletNotFound) {
			utils.WriteError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Top up successful", res)
}

func (h *TransactionHandler) Transfer(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	var req dto.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	res, err := h.transactionService.Transfer(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, service.ErrSelfTransfer) {
			utils.WriteError(c, http.StatusUnprocessableEntity, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrInsufficientBalance) {
			utils.WriteError(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrTargetWalletNotFound) || errors.Is(err, service.ErrWalletNotFound) {
			utils.WriteError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Transfer successful", res)
}

func (h *TransactionHandler) GetTransactions(c *gin.Context) {
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

	var txnType *string
	if t := c.Query("type"); t != "" {
		txnType = &t
	}

	res, err := h.transactionService.GetTransactions(c.Request.Context(), userID, txnType, pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Transactions retrieved successfully", res)
}

func (h *TransactionHandler) GetTransactionDetail(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	txIDStr := c.Param("transactionId")
	txID, err := uuid.Parse(txIDStr)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Invalid transaction ID format", nil)
		return
	}

	res, err := h.transactionService.GetTransactionDetail(c.Request.Context(), userID, txID)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			utils.WriteError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			utils.WriteError(c, http.StatusForbidden, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Transaction details retrieved successfully", res)
}

func (h *TransactionHandler) GetTransactionsAdmin(c *gin.Context) {
	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	var txnType *string
	if t := c.Query("type"); t != "" {
		txnType = &t
	}

	var txnStatus *string
	if s := c.Query("status"); s != "" {
		txnStatus = &s
	}

	res, err := h.transactionService.GetAllTransactions(c.Request.Context(), txnType, txnStatus, pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Transactions retrieved successfully", res)
}

func (h *TransactionHandler) GetTransactionDetailAdmin(c *gin.Context) {
	txIDStr := c.Param("transactionId")
	txID, err := uuid.Parse(txIDStr)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Invalid transaction ID format", nil)
		return
	}

	res, err := h.transactionService.GetTransactionDetailAdmin(c.Request.Context(), txID)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			utils.WriteError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Transaction details retrieved successfully", res)
}
