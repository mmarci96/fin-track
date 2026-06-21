// Package apperr defines the application's error model. It gives errors a
// stable code, an HTTP status, and a client-safe message, while preserving the
// wrapped cause (with its origin stack trace, captured by cockroachdb/errors)
// for logging.
//
// Convention:
//   - Wrap at the origin:   errors.Wrapf(err, "create receipt id=%d", id)
//   - Classify at the edge:  return apperr.Internal("could not save receipt", err)
//   - Log once, at the HTTP boundary, with LogArgs.
package apperr

import (
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
)

// AppError carries the information needed to both respond to a client safely
// and log the full failure internally.
type AppError struct {
	// Code is a stable, machine-readable identifier (e.g. "DB_WRITE_FAILED").
	Code string
	// Status is the HTTP status to return to the client.
	Status int
	// Public is a message safe to expose to the client. It must never contain
	// internal details.
	Public string
	// cause is the wrapped underlying error, kept for logging only.
	cause error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return e.Code + ": " + e.cause.Error()
	}
	return e.Code + ": " + e.Public
}

// Unwrap exposes the cause so errors.Is/As and stack rendering work across the
// chain.
func (e *AppError) Unwrap() error { return e.cause }

// new builds an AppError, ensuring the cause carries a stack trace.
func newAppError(code string, status int, public string, cause error) *AppError {
	if cause != nil {
		// WithStack is a no-op if a stack is already attached, so this is safe
		// even when the caller already wrapped.
		cause = errors.WithStack(cause)
	}
	return &AppError{Code: code, Status: status, Public: public, cause: cause}
}

// Constructors for the common HTTP classes. Add codes as the app grows.

func BadRequest(public string, cause error) *AppError {
	return newAppError("BAD_REQUEST", http.StatusBadRequest, public, cause)
}

func NotFound(public string, cause error) *AppError {
	return newAppError("NOT_FOUND", http.StatusNotFound, public, cause)
}

func Internal(public string, cause error) *AppError {
	return newAppError("INTERNAL", http.StatusInternalServerError, public, cause)
}

// Coded lets callers attach a specific domain code while still choosing the
// status/public message.
func Coded(code string, status int, public string, cause error) *AppError {
	return newAppError(code, status, public, cause)
}

// Sentinel errors for known domain conditions. Compare with errors.Is.
var (
	// ErrNoMerchantMatch means OCR text could not be matched to a known
	// merchant. The OCR handler treats this as a soft outcome (200 + warning)
	// rather than a failure.
	ErrNoMerchantMatch = errors.New("no merchant matched from db")
)

// As extracts an *AppError from anywhere in err's chain, if present.
func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

// HTTPResponse maps any error to the (status, code, publicMessage) that should
// be returned to a client. Unclassified errors collapse to a generic 500 so we
// never leak internals.
func HTTPResponse(err error) (status int, code, public string) {
	if ae, ok := As(err); ok {
		return ae.Status, ae.Code, ae.Public
	}
	return http.StatusInternalServerError, "INTERNAL", "internal server error"
}

// LogArgs returns slog key/value pairs describing err for structured logging,
// including the full wrapped chain with origin stack traces. Intended to be
// spread into a single log call at the boundary: logger.Error("...", LogArgs(err)...).
func LogArgs(err error) []any {
	if err == nil {
		return nil
	}
	args := []any{
		"error", err.Error(),
		// %+v renders the full cause chain with stack traces captured at origin.
		"error_detail", fmt.Sprintf("%+v", err),
	}
	if ae, ok := As(err); ok {
		args = append(args, "error_code", ae.Code)
	}
	return args
}
