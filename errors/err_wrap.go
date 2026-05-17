package errors

import (
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/domains"
)

// Wrap wraps an error with a message prefix.
//   - When to use: on error return paths.
//   - What it does: combines WithStack().
//   - How to access the detail: Error(), regular Go formatting.
func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

// Wrapf wraps an error with a formatted message prefix.
// If the format is empty, no prefix is added, but the extra arguments are still processed for reportable strings.
//   - When to use: on error return paths.
//   - What it does: combines WithStack().
//   - How to access the detail: Error(), regular Go formatting.
func Wrapf(err error, format string, args ...any) error {
	return errors.Wrapf(err, format, args...)
}

// WithStack annotates an error with a stack trace of depth 1.
//   - When to use: need to capture the stack trace of an error.
//   - What it does: captures (efficiently) a stack trace.
//   - How to access the detail: format with %+v.
//
// Note: Stack traces do not appear in the main error message returned with Error().
func WithStack(err error) error {
	return WithStackDepth(err, 1)
}

// WithStack annotates an error with a stack trace of depth N.
//   - When to use: need to capture the stack trace of an error.
//   - What it does: captures (efficiently) a stack trace.
//   - How to access the detail: format with %+v.
//
// Note: Stack traces do not appear in the main error message returned with Error().
func WithStackDepth(err error, depth int) error {
	return errors.WithStackDepth(err, depth)
}

// WithDetail annotates an error with a user-facing detail with contextual information.
//   - When to use: need to embark a message string to output when the error is presented to a developer.
//   - What it does: captures detail strings.
//   - How to access the detail: format with %+v.
//
// Note: Details does not appear in the main error message returned with Error().
func WithDetail(err error, detail string) error {
	return errors.WithDetail(err, detail)
}

// WithHint annotates an error with a user-facing detail as a suggestion for action to take.
//   - when to use: need to embark a message string to output when the error is presented to an end user.
//   - What it does: captures hint strings.
//   - How to access the detail: format with %+v.
//
// Note: Hints does not appear in the main error message returned with Error().
func WithHint(err error, hint string) error {
	return errors.WithHint(err, hint)
}

// WithDomain annotates an error with an origin package.
//   - When to use: at package boundaries.
//   - What it does: captures the identity of the error domain.
//   - Can be asserted with errors.EnsureNotInDomain(), errors.NotInDomain().
//   - How to access the detail: format with %+v.
//
// Note: Domains does not appear in the main error message returned with Error().
func WithDomain(err error, domain string) error {
	return errors.WithDomain(err, domains.NamedDomain(domain))
}
