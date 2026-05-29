package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("accessToken")
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, "Unauthorized: please login first", nil)
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString, secret)
		if err != nil {
			utils.WriteError(c, http.StatusUnauthorized, "Unauthorized: invalid or expired token", nil)
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RoleGuard(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			utils.WriteError(c, http.StatusUnauthorized, "Unauthorized: role not found", nil)
			c.Abort()
			return
		}

		role, ok := roleVal.(string)
		if !ok {
			utils.WriteError(c, http.StatusUnauthorized, "Unauthorized: invalid role type", nil)
			c.Abort()
			return
		}

		allowed := false
		for _, r := range allowedRoles {
			if r == role {
				allowed = true
				break
			}
		}

		if !allowed {
			utils.WriteError(c, http.StatusForbidden, "Forbidden: you do not have permission to access this resource", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, errors.New("user ID not found in context")
	}
	id, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user ID in context is not a valid UUID")
	}
	return id, nil
}

func GetEmailFromContext(c *gin.Context) (string, error) {
	val, exists := c.Get("email")
	if !exists {
		return "", errors.New("email not found in context")
	}
	email, ok := val.(string)
	if !ok {
		return "", errors.New("email in context is not a valid string")
	}
	return email, nil
}
