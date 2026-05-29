package utils

import (
	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	ErrorCode int         `json:"error_code"`
	Errors    interface{} `json:"errors,omitempty"`
}

func WriteSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func WriteError(c *gin.Context, statusCode int, message string, errors interface{}) {
	c.JSON(statusCode, ErrorResponse{
		Success:   false,
		Message:   message,
		ErrorCode: statusCode,
		Errors:    errors,
	})
}
