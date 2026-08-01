// Package apperr maps domain errors to HTTP status codes without leaking
// internal details to the client.
package apperr

import (
	"errors"
	"net/http"

	"erdinhrmwn/bangunin/internal/domain/errs"
)

// AppError is the error type handlers should return; it carries everything
// pkg/response needs to render the error envelope.
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *AppError) Error() string { return e.Message }

func New(code, message string, httpStatus int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus}
}

// From maps a domain error (or any error) to an AppError. Unmapped errors
// become a generic 500 with no internal detail exposed.
func From(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, errs.ErrNotFound):
		return New("NOT_FOUND", "Data not found", http.StatusNotFound)
	case errors.Is(err, errs.ErrValidation):
		return New("VALIDATION_ERROR", "Invalid data", http.StatusUnprocessableEntity)
	case errors.Is(err, errs.ErrUnauthorized):
		return New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
	case errors.Is(err, errs.ErrForbidden):
		return New("FORBIDDEN", "Access denied", http.StatusForbidden)
	case errors.Is(err, errs.ErrConflict):
		return New("CONFLICT", "Data conflict", http.StatusConflict)
	default:
		return New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	}
}
