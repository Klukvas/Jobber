package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Data interface{} `json:"data"`
}

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, statusCode int, errorCode, errorMessage string) {
	c.JSON(statusCode, ErrorResponse{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	})
}

// RespondWithSuccess sends a standardized success response
func RespondWithSuccess(c *gin.Context, statusCode int, data interface{}) {
	if data == nil {
		c.JSON(statusCode, gin.H{})
		return
	}
	c.JSON(statusCode, SuccessResponse{Data: data})
}

// RespondWithData sends data directly without wrapping
func RespondWithData(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// Health response structure
type HealthResponse struct {
	Status   string            `json:"status"`
	Version  string            `json:"version"`
	Services map[string]string `json:"services"`
}

// RespondWithHealth sends a health check response
func RespondWithHealth(c *gin.Context, services map[string]string) {
	status := "healthy"
	for _, serviceStatus := range services {
		if serviceStatus != "up" {
			status = "degraded"
			break
		}
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:   status,
		Version:  "1.0.0",
		Services: services,
	})
}
