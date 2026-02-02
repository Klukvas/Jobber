package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
	DefaultOffset = 0
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Limit  int
	Offset int
}

// PaginationMeta represents pagination metadata in responses
type PaginationMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Items      interface{}    `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// ParsePaginationParams extracts and validates pagination parameters from request
func ParsePaginationParams(c *gin.Context) (*PaginationParams, error) {
	limit := DefaultLimit
	offset := DefaultOffset

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 0 {
			return nil, ErrInvalidPaginationParams
		}
		limit = parsedLimit
		// Clamp to max
		if limit > MaxLimit {
			limit = MaxLimit
		}
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			return nil, ErrInvalidPaginationParams
		}
		offset = parsedOffset
	}

	return &PaginationParams{
		Limit:  limit,
		Offset: offset,
	}, nil
}

// RespondWithPagination sends a paginated response
func RespondWithPagination(c *gin.Context, statusCode int, items interface{}, limit, offset, total int) {
	c.JSON(statusCode, PaginatedResponse{
		Items: items,
		Pagination: PaginationMeta{
			Limit:  limit,
			Offset: offset,
			Total:  total,
		},
	})
}
