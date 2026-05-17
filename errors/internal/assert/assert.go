package assert

import (
	"context"
	"errors"
	"fmt"

	cerrors "github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
	"github.com/gogo/protobuf/proto"
)

type withAssertionFailure struct {
	cause error
}

var _ error = (*withAssertionFailure)(nil)
var _ fmt.Formatter = (*withAssertionFailure)(nil)
var _ errbase.SafeFormatter = (*withAssertionFailure)(nil)

func WithAssertionFailure(err error) error {
	if err == nil {
		return nil
	}
	return &withAssertionFailure{cause: err}
}

func HasAssertionFailure(err error) bool {
	if cerrors.HasAssertionFailure(err) {
		return true
	}

	var wrapped *withAssertionFailure
	return errors.As(err, &wrapped)
}

func IsAssertionFailure(err error) bool {
	if cerrors.IsAssertionFailure(err) {
		return true
	}

	_, ok := err.(*withAssertionFailure)
	return ok
}

func (w *withAssertionFailure) Error() string { return w.cause.Error() }
func (w *withAssertionFailure) Cause() error  { return w.cause }
func (w *withAssertionFailure) Unwrap() error { return w.cause }

func (w *withAssertionFailure) Format(s fmt.State, verb rune) { errbase.FormatError(w, s, verb) }

func (w *withAssertionFailure) SafeFormatError(p errbase.Printer) error {
	if p.Detail() {
		p.Printf("assertion failure")
	}
	return w.cause
}

func decodeAssertFailure(
	_ context.Context, cause error, _ string, _ []string, _ proto.Message,
) error {
	return &withAssertionFailure{cause: cause}
}

func init() {
	errbase.RegisterWrapperDecoder(errbase.GetTypeKey((*withAssertionFailure)(nil)), decodeAssertFailure)
}
