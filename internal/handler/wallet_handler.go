package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"marcceljanara/wallet-ledger-service/internal/middleware"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type WalletHandler struct {
	walletService service.WalletService
}

func NewWalletHandler(walletService service.WalletService) *WalletHandler {
	return &WalletHandler{
		walletService: walletService,
	}
}

func (h *WalletHandler) GetMyWallet(c *gin.Context) {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	res, err := h.walletService.GetWalletByUserID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrWalletNotFound) {
			utils.WriteError(c, http.StatusNotFound, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Wallet retrieved successfully", res)
}
