// Package domain contains the core domain models and types.
package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure cases.
var (
	// ErrEmptyLog indicates the log content is empty or whitespace only.
	ErrEmptyLog = errors.New("log content is empty")

	// ErrLogTooLarge indicates the log exceeds the maximum allowed size.
	ErrLogTooLarge = errors.New("log content exceeds maximum size")

	// ErrAITimeout indicates the AI service did not respond in time.
	ErrAITimeout = errors.New("AI service timeout")

	// ErrAIUnavailable indicates the AI service is not available.
	ErrAIUnavailable = errors.New("AI service unavailable")

	// ErrInvalidAIResponse indicates the AI response failed validation.
	ErrInvalidAIResponse = errors.New("invalid AI response format")

	// ErrRateLimited indicates too many requests were made.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrInvalidConfig indicates invalid configuration.
	ErrInvalidConfig = errors.New("invalid configuration")
)

// AnalysisError wraps an error with additional context.
type AnalysisError struct {
	// Op is the operation that failed.
	Op string

	// Err is the underlying error.
	Err error

	// Retryable indicates if the operation can be retried.
	Retryable bool
}

// Error implements the error interface.
func (e *AnalysisError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *AnalysisError) Unwrap() error {
	return e.Err
}

// WrapError creates a new AnalysisError with context.
func WrapError(op string, err error, retryable bool) *AnalysisError {
	return &AnalysisError{
		Op:        op,
		Err:       err,
		Retryable: retryable,
	}
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var ae *AnalysisError
	if errors.As(err, &ae) {
		return ae.Retryable
	}
	return false
}
