package script

import "fmt"

// ErrorCategory represents the type of error that occurred
type ErrorCategory int

const (
	// ScriptError indicates an error with the script itself
	ScriptError ErrorCategory = iota
	// HookError indicates an error in a script hook
	HookError
	// ValidationError indicates a configuration validation error
	ValidationError
	// ExecutionError indicates an error during script execution
	ExecutionError
)

// Error represents a script-related error with context and recovery suggestions
type Error struct {
	Category ErrorCategory
	Message  string
	Details  string
	Fix      string
}

// Error implements the error interface
func (e *Error) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Category.String(), e.Message)
	if e.Details != "" {
		msg += fmt.Sprintf("\nDetails: %s", e.Details)
	}
	if e.Fix != "" {
		msg += fmt.Sprintf("\nTo fix: %s", e.Fix)
	}
	return msg
}

// String returns a string representation of the error category
func (c ErrorCategory) String() string {
	switch c {
	case ScriptError:
		return "Script Error"
	case HookError:
		return "Hook Error"
	case ValidationError:
		return "Validation Error"
	case ExecutionError:
		return "Execution Error"
	default:
		return "Unknown Error"
	}
}

// NewScriptError creates a new script error
func NewScriptError(message string, details ...string) *Error {
	e := &Error{
		Category: ScriptError,
		Message:  message,
	}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

// NewHookError creates a new hook error
func NewHookError(message string, details ...string) *Error {
	e := &Error{
		Category: HookError,
		Message:  message,
	}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details ...string) *Error {
	e := &Error{
		Category: ValidationError,
		Message:  message,
	}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

// NewExecutionError creates a new execution error
func NewExecutionError(message string, details ...string) *Error {
	e := &Error{
		Category: ExecutionError,
		Message:  message,
	}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

// WithFix adds a fix suggestion to the error
func (e *Error) WithFix(fix string) *Error {
	e.Fix = fix
	return e
}
