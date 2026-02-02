package http

import "errors"

var (
	// ErrInvalidPaginationParams is returned when pagination parameters are invalid
	ErrInvalidPaginationParams = errors.New("invalid pagination parameters")
)
