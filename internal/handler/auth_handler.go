package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type AuthHandler struct {
	authService service.AuthService
	jwtExpiry   int // maxAge in seconds
	secure      bool
}

func NewAuthHandler(authService service.AuthService, jwtExpiry int, secure bool) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtExpiry:   jwtExpiry,
		secure:      secure,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	res, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmailConflict) {
			utils.WriteError(c, http.StatusConflict, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusCreated, "User registered successfully", res)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	res, token, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			utils.WriteError(c, http.StatusUnauthorized, err.Error(), nil)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("accessToken", token, h.jwtExpiry, "/", "", h.secure, true)

	utils.WriteSuccess(c, http.StatusOK, "Login successful", res)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("accessToken", "", -1, "/", "", h.secure, true)

	utils.WriteSuccess(c, http.StatusOK, "Logout successful", nil)
}

func (h *AuthHandler) GetUsersAdmin(c *gin.Context) {
	var pagination dto.PaginationRequest
	if err := c.ShouldBindQuery(&pagination); err != nil {
		utils.WriteError(c, http.StatusBadRequest, "Validation failed", dto.FormatValidationErrors(err))
		return
	}

	res, err := h.authService.GetAllUsers(c.Request.Context(), pagination)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	utils.WriteSuccess(c, http.StatusOK, "Users retrieved successfully", res)
}
