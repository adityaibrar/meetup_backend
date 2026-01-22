package models

import (
	"time"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     interface{} `json:"error,omitempty"`
	Meta      interface{} `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ValidationErrors represents validation error response
type ValidationErrors struct {
	Errors []ErrorDetail `json:"errors"`
}

// SuccessResponse creates a standardized success response
func SuccessResponse(message string, data interface{}, meta interface{}) APIResponse {
	return APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now(),
	}
}

// ErrorResponse creates a standardized error response
func ErrorResponse(message string, error interface{}) APIResponse {
	return APIResponse{
		Success:   false,
		Message:   message,
		Error:     error,
		Timestamp: time.Now(),
	}
}

// NewPaginationMeta creates pagination metadata
func NewPaginationMeta(page, limit int, total int64) PaginationMeta {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return PaginationMeta{
		CurrentPage: page,
		PerPage:     limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
}
