package validate

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/errors/errorspb"
)

// testPrinter implements errbase.Printer for unit tests.
type testPrinter struct {
	b      strings.Builder
	detail bool
}

func (p *testPrinter) Print(args ...interface{})                 { fmt.Fprint(&p.b, args...) }
func (p *testPrinter) Printf(format string, args ...interface{}) { fmt.Fprintf(&p.b, format, args...) }
func (p *testPrinter) Detail() bool                              { return p.detail }

func TestNewValidationError(t *testing.T) {
	err := ValidationFailedf("email", "alice@example.com", "required")
	if _, ok := err.(*ValidationError); !ok {
		t.Fatalf("expected validationError, got %T", err)
	}
}

// TestEncodeDecodeValidationError verifies encoding and decoding of our
// validationError via the registered encoder/decoder helpers.
func TestEncodeDecodeValidationError(t *testing.T) {
	v := new(ValidationError{field: "email", value: "alice@example.com", rule: "required"})

	msg, details, payload := encodeValidationError(context.Background(), v)
	if msg != v.Error() {
		t.Fatalf("unexpected msg: %q, want %q", msg, v.Error())
	}

	wantDetails := v.SafeDetails()
	if len(details) != len(wantDetails) {
		t.Fatalf("unexpected safe details length: %d vs %d", len(details), len(wantDetails))
	}

	for i := range details {
		if details[i] != wantDetails[i] {
			t.Fatalf("detail[%d]=%q, want %q", i, details[i], wantDetails[i])
		}
	}

	sp, ok := payload.(*errorspb.StringPayload)
	if !ok {
		t.Fatalf("payload type mismatch: %T", payload)
	}
	if sp.Msg != "alice@example.com" {
		t.Fatalf("payload msg = %q, want %q", sp.Msg, "alice@example.com")
	}

	// now decode back
	got := decodeValidationError(context.Background(), msg, details, payload)
	ve, ok := got.(*ValidationError)
	if !ok {
		t.Fatalf("decoded error has wrong type: %T", got)
	}
	if ve.field != v.field || ve.rule != v.rule || ve.value != v.value {
		t.Fatalf("decoded value mismatch: %+v vs %+v", ve, v)
	}
}

func TestEncodeNonValidationError(t *testing.T) {
	// When the error is not a validationError, the encoder should return
	// the original message and nil details/payload.
	// create a simple error value
	e := errorString("plain")

	msg, details, payload := encodeValidationError(context.Background(), e)
	if msg != e.Error() {
		t.Fatalf("msg = %q, want %q", msg, e.Error())
	}
	if details != nil {
		t.Fatalf("expected nil details, got %v", details)
	}
	if payload != nil {
		t.Fatalf("expected nil payload, got %T", payload)
	}
}

func TestFormatValidationError(t *testing.T) {
	v := new(ValidationError{field: "username", value: "bob", rule: "minLength:3"})

	// %s and %q should produce the same output as Error()
	s := v.Error()
	if fmt.Sprintf("%s", v) != s {
		t.Fatalf("%%s format mismatch: %q vs %q", fmt.Sprintf("%s", v), s)
	}
	if fmt.Sprintf("%q", v) != s {
		t.Fatalf("%%q format mismatch: %q vs %q", fmt.Sprintf("%q", v), s)
	}
	if fmt.Sprintf("%v", v) != s {
		t.Fatalf("%%v format mismatch: %q vs %q", fmt.Sprintf("%q", v), s)
	}

	// %+v should include all details
	detailStr := fmt.Sprintf("%+v", v)
	if !strings.Contains(detailStr, "field: \"username\"") ||
		!strings.Contains(detailStr, "value: bob") ||
		!strings.Contains(detailStr, "rule: \"minLength:3\"") {
		t.Fatalf("%%+v format missing details: %s", detailStr)
	}
}

func TestSafeFormatErrorPrinter(t *testing.T) {
	v := &ValidationError{field: "username", value: "bob", rule: "minLength:3"}

	// when Detail() is false, SafeFormatError must not print the value
	p1 := &testPrinter{detail: false}
	next := v.SafeFormatError(p1)
	if next != nil {
		t.Fatalf("expected nil next, got %v", next)
	}
	if p1.b.Len() != 0 {
		t.Fatalf("expected no output when Detail() is false, got %q", p1.b.String())
	}

	// when Detail() is true, SafeFormatError should include the value
	p2 := &testPrinter{detail: true}
	_ = v.SafeFormatError(p2)
	if !strings.Contains(p2.b.String(), "value=bob") {
		t.Fatalf("expected detailed output to include value, got %q", p2.b.String())
	}
}

// errorString is a tiny helper type to create an error from a string value.
type errorString string

func (s errorString) Error() string { return string(s) }
