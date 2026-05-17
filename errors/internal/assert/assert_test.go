package assert

import (
	"context"
	stdErrors "errors"
	"fmt"
	"testing"

	cerrors "github.com/cockroachdb/errors"
)

func TestWithAssertionFailure(t *testing.T) {
	base := stdErrors.New("boom")
	wrapped := WithAssertionFailure(base)
	if wrapped == nil {
		t.Fatalf("WithAssertionFailure returned nil")
	}
	if !IsAssertionFailure(wrapped) {
		t.Fatalf("wrapped error was not recognized as an assertion failure")
	}
	if !HasAssertionFailure(wrapped) {
		t.Fatalf("wrapped error was not found in the error chain")
	}
	if got := wrapped.Error(); got != base.Error() {
		t.Fatalf("wrapped Error() = %q, want %q", got, base.Error())
	}
	if !stdErrors.Is(wrapped, base) {
		t.Fatalf("wrapped error does not unwrap to base error")
	}
	if got := fmt.Sprintf("%+v", wrapped); got == "" {
		t.Fatalf("expected detailed formatting to produce output")
	}
}

func TestWithAssertionFailureNil(t *testing.T) {
	if got := WithAssertionFailure(nil); got != nil {
		t.Fatalf("WithAssertionFailure(nil) = %v, want nil", got)
	}
}

func TestDecodeAssertFailure(t *testing.T) {
	base := cerrors.New("base")
	got := decodeAssertFailure(context.Background(), base, "", nil, nil)
	if _, ok := got.(*withAssertionFailure); !ok {
		t.Fatalf("decoded value has wrong type: %T", got)
	}
}

func TestHasAssertionFailureWithUpstreamMarker(t *testing.T) {
	base := cerrors.New("x")
	wrapped := cerrors.WithAssertionFailure(base)
	if !HasAssertionFailure(wrapped) {
		t.Fatalf("HasAssertionFailure did not detect upstream marker")
	}
	if !IsAssertionFailure(wrapped) {
		t.Fatalf("IsAssertionFailure did not detect upstream marker")
	}
}
