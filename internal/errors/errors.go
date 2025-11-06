package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// MCPError represents a structured error with context
type MCPError struct {
	Component string
	Operation string
	Message   string
	Cause     error
	File      string
	Line      int
}

// Error implements the error interface
func (e *MCPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s.%s] %s: %v", e.Component, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s.%s] %s", e.Component, e.Operation, e.Message)
}

// Unwrap returns the underlying cause
func (e *MCPError) Unwrap() error {
	return e.Cause
}

// GetLocation returns the file and line where the error occurred
func (e *MCPError) GetLocation() string {
	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s:%d", e.File, e.Line)
	}
	return "unknown"
}

// New creates a new MCPError
func New(component, operation, message string) *MCPError {
	file, line := getCallerInfo(2)
	return &MCPError{
		Component: component,
		Operation: operation,
		Message:   message,
		File:      file,
		Line:      line,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, component, operation, message string) *MCPError {
	if err == nil {
		return nil
	}

	file, line := getCallerInfo(2)
	return &MCPError{
		Component: component,
		Operation: operation,
		Message:   message,
		Cause:     err,
		File:      file,
		Line:      line,
	}
}

// getCallerInfo returns the file and line number of the caller
func getCallerInfo(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}

	// Get just the filename, not the full path
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return file, line
}

// Predefined error creators for different components
var (
	// Database errors
	DatabaseError = func(operation, message string) *MCPError {
		return New("database", operation, message)
	}
	DatabaseWrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "database", operation, message)
	}

	// Tool errors
	ToolError = func(operation, message string) *MCPError {
		return New("tool", operation, message)
	}
	ToolWrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "tool", operation, message)
	}

	// Resource errors
	ResourceError = func(operation, message string) *MCPError {
		return New("resource", operation, message)
	}
	ResourceWrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "resource", operation, message)
	}

	// Server errors
	ServerError = func(operation, message string) *MCPError {
		return New("server", operation, message)
	}
	ServerWrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "server", operation, message)
	}

	// LLM errors
	LLMError = func(operation, message string) *MCPError {
		return New("llm", operation, message)
	}
	LLMWrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "llm", operation, message)
	}

	// S3 errors
	S3Error = func(operation, message string) *MCPError {
		return New("s3", operation, message)
	}
	S3Wrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "s3", operation, message)
	}

	// Loki errors
	LokiError = func(operation, message string) *MCPError {
		return New("loki", operation, message)
	}
	LokiWrap = func(err error, operation, message string) *MCPError {
		return Wrap(err, "loki", operation, message)
	}
)

// IsComponentError checks if an error belongs to a specific component
func IsComponentError(err error, component string) bool {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Component == component
	}
	return false
}

// GetErrorChain returns all errors in the chain
func GetErrorChain(err error) []error {
	var chain []error
	for err != nil {
		chain = append(chain, err)
		if mcpErr, ok := err.(*MCPError); ok {
			err = mcpErr.Cause
		} else {
			break
		}
	}
	return chain
}
