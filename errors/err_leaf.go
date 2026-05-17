package errors

import (
	"github.com/GabrielNunesIT/go-libs/errors/internal/cache"
	"github.com/GabrielNunesIT/go-libs/errors/internal/config"
	"github.com/GabrielNunesIT/go-libs/errors/internal/startup"
	"github.com/GabrielNunesIT/go-libs/errors/internal/validate"
	"github.com/cockroachdb/errors"
)

// New creates a new leaf error with the given message.
//   - When to use: common error cases.
//   - What it does: creates a new error with the message, and also captures the stack trace at point of call and redacts the provided message for safe reporting.
//   - How to access the detail: Error(), regular Go formatting.
func New(msg string) error {
	return errors.New(msg)
}

// New creates a new leaf error with a formatted message.
//   - When to use: common error cases.
//   - What it does: creates a new error with the message, and also captures the stack trace at point of call and redacts the provided message for safe reporting.
//   - How to access the detail: Error(), regular Go formatting.
func Newf(format string, args ...any) error {
	return errors.Newf(format, args...)
}

// AssertionFailedf signals an assertion failure / programming error.
//   - When to use: when an invariant is violated; when an unreachable code path is reached.
//   - What it does: creates a new error with the message, also captures the stack trace at point of call, redacts the provided strings for safe reporting, prepares a hint to inform a human user.
//   - Can be asserted with assert.IsAssertionFailure()/assert.HasAssertionFailure()
//   - How to access the detail: format with %+v.
func AssertionFailedf(format string, args ...any) error {
	return errors.AssertionFailedf(format, args...)
}

// ValidationFailedf creates a new validation error with the given field, value, and rule.
//   - When to use: when validating user input or data and a validation rule is violated.
//   - What it does: creates a new validationError with the provided field, value, and rule. The error message includes the field and rule, while the value is included as a payload for safe reporting.
//   - How to access the detail: Error() for the message; the field and rule are included in the error message, and the value can be accessed via error inspection (e.g., errors.As).
func ValidationFailedf(field string, value any, rule string) error {
	return validate.ValidationFailedf(field, value, rule)
}

// CacheErrorf creates a new cache operation error.
//   - When to use: cache operations fail but are not necessarily fatal.
//   - What it does: creates a new cacheError with the provided operation, key, and message.
//   - How to access the detail: Error() for the message; operation and key are included in the message, and details can be accessed via error inspection (e.g., errors.As).
func CacheErrorf(operation string, key string, message string) error {
	return cache.CacheErrorf(operation, key, message)
}

// InitializationErrorf creates a new initialization error.
//   - When to use: during startup/bootstrap/setup phases when a component fails to initialize.
//   - What it does: creates a new initializationError with the provided component name and message.
//   - How to access the detail: Error() for the message; component is included in the message, and details can be accessed via error inspection (e.g., errors.As).
func InitializationErrorf(component string, message string) error {
	return startup.InitializationErrorf(component, message)
}

// ConfigurationErrorf creates a new configuration error.
//   - When to use: when configuration is invalid, missing, malformed, or inconsistent.
//   - What it does: creates a new configurationError with the provided field, issue type, and message.
//   - How to access the detail: Error() for the message; field and issue are included in the message, and details can be accessed via error inspection (e.g., errors.As).
func ConfigurationErrorf(field string, issue string, message string) error {
	return config.ConfigurationErrorf(field, issue, message)
}
