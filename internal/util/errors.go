// Package util contains shared formatting and error helpers used across
// internal packages.
package util

import "fmt"

// UsageError marks invalid CLI input that should return a usage exit code.
type UsageError struct {
	Message string
}

func (e *UsageError) Error() string {
	return e.Message
}

// NewUsageError creates an error that signals invalid user input.
func NewUsageError(format string, args ...any) error {
	return &UsageError{Message: fmt.Sprintf(format, args...)}
}
