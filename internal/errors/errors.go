// internal/errors/errors.go
package errors

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ErrorType represents different categories of errors
type ErrorType int

const (
	// Error types
	TypeGeneral ErrorType = iota
	TypeSecurity
	TypeFileOperation
	TypeKeyManagement
	TypeNetwork
	TypeConfig
)

// AppError represents an application error with context
type AppError struct {
	Type    ErrorType
	Message string
	Cause   error
	Data    map[string]interface{}
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithData adds context data to the error
func (e *AppError) WithData(key string, value interface{}) *AppError {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// New creates a new application error
func New(errType ErrorType, message string) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Type:    errType,
		Message: message,
		Cause:   err,
	}
}

// FormatErrorForDisplay formats an error for display in the UI
func FormatErrorForDisplay(err error) string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Padding(1)

	if err == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Error: ")
	builder.WriteString(err.Error())

	// If it's our AppError, add additional context
	if appErr, ok := err.(*AppError); ok {
		// Add type-specific context
		switch appErr.Type {
		case TypeSecurity:
			builder.WriteString("\n\nThis is a security-related error. Please ensure you have the correct permissions.")
		case TypeFileOperation:
			builder.WriteString("\n\nThis error occurred during a file operation. Please check file paths and permissions.")
		case TypeKeyManagement:
			builder.WriteString("\n\nThis error occurred during key management. Your keys may be corrupted or inaccessible.")
		}

		// Add any context data
		if appErr.Data != nil && len(appErr.Data) > 0 {
			builder.WriteString("\n\nAdditional information:")
			for k, v := range appErr.Data {
				builder.WriteString(fmt.Sprintf("\n- %s: %v", k, v))
			}
		}
	}

	return errorStyle.Render(builder.String())
}

// IsSecurityError returns true if the error is security-related
func IsSecurityError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == TypeSecurity
	}
	return false
}

// IsFileError returns true if the error is file-related
func IsFileError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == TypeFileOperation
	}
	return false
}

// IsKeyManagementError returns true if the error is key-related
func IsKeyManagementError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == TypeKeyManagement
	}
	return false
}

