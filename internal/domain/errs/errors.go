// Package errs defines domain-level sentinel errors shared across usecases.
// pkg/apperr maps these to HTTP status codes at the delivery boundary.
package errs

import "errors"

var (
	ErrNotFound     = errors.New("resource not found")
	ErrValidation   = errors.New("validation failed")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
)
