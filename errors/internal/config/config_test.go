package config

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/errors/errorspb"
)

func TestNewConfigurationError(t *testing.T) {
	err := ConfigurationErrorf("database_url", "missing", "environment variable not set")
	if _, ok := err.(*configurationError); !ok {
		t.Fatalf("expected configurationError, got %T", err)
	}
}

func TestConfigurationErrorFormatting(t *testing.T) {
	err := ConfigurationErrorf("port", "invalid", "must be a positive integer")
	errStr := err.Error()
	if !strings.Contains(errStr, "configuration error") {
		t.Fatalf("error message missing 'configuration error': %q", errStr)
	}
	if !strings.Contains(errStr, "port") {
		t.Fatalf("error message missing field: %q", errStr)
	}
	if !strings.Contains(errStr, "invalid") {
		t.Fatalf("error message missing issue: %q", errStr)
	}
}

func TestConfigurationErrorSafeDetails(t *testing.T) {
	err := ConfigurationErrorf("timeout", "malformed", "duration parsing failed: invalid format")
	ce, ok := err.(*configurationError)
	if !ok {
		t.Fatalf("type assertion failed")
	}

	details := ce.SafeDetails()
	if len(details) != 3 {
		t.Fatalf("expected 3 safe details, got %d", len(details))
	}

	detailStr := strings.Join(details, "|")
	if !strings.Contains(detailStr, "field: timeout") {
		t.Fatalf("safe details missing field: %v", details)
	}
	if !strings.Contains(detailStr, "issue: malformed") {
		t.Fatalf("safe details missing issue: %v", details)
	}
}

func TestEncodeDecodeConfigurationError(t *testing.T) {
	ce := &configurationError{field: "auth_key", issue: "missing", message: "no auth key provided"}

	msg, details, payload := encodeConfigurationError(context.Background(), ce)
	if msg != ce.Error() {
		t.Fatalf("unexpected msg: %q, want %q", msg, ce.Error())
	}

	wantDetails := ce.SafeDetails()
	if len(details) != len(wantDetails) {
		t.Fatalf("unexpected safe details length: %d vs %d", len(details), len(wantDetails))
	}

	sp, ok := payload.(*errorspb.StringPayload)
	if !ok {
		t.Fatalf("payload type mismatch: %T", payload)
	}
	if sp.Msg != "no auth key provided" {
		t.Fatalf("payload msg = %q, want %q", sp.Msg, "no auth key provided")
	}

	// decode back
	got := decodeConfigurationError(context.Background(), msg, details, payload)
	ce2, ok := got.(*configurationError)
	if !ok {
		t.Fatalf("decoded error has wrong type: %T", got)
	}
	if ce2.field != ce.field || ce2.issue != ce.issue || ce2.message != ce.message {
		t.Fatalf("decoded value mismatch: %+v vs %+v", ce2, ce)
	}
}

func TestConfigurationErrorVerboseFormat(t *testing.T) {
	err := ConfigurationErrorf("log_level", "invalid", "unknown level")
	formatted := fmt.Sprintf("%+v", err)
	if formatted == "" {
		t.Fatalf("expected verbose formatting to produce output")
	}
	if !strings.Contains(formatted, "configurationError") {
		t.Fatalf("verbose format missing type name: %q", formatted)
	}
}

// Additional tests merged from config_extra_test.go
// testPrinter implements errbase.Printer for unit tests.
type testPrinter struct {
	b      string
	detail bool
}

func (p *testPrinter) Print(args ...interface{})                 { p.b += fmt.Sprint(args...) }
func (p *testPrinter) Printf(format string, args ...interface{}) { p.b += fmt.Sprintf(format, args...) }
func (p *testPrinter) Detail() bool                              { return p.detail }

func TestConfigSafeFormatError(t *testing.T) {
	e := &configurationError{field: "f", issue: "i", message: "m"}
	p := &testPrinter{detail: true}
	next := e.SafeFormatError(p)
	if next != nil {
		t.Fatalf("expected nil next, got %v", next)
	}

	if fmt.Sprintf("%s", e) == "" || fmt.Sprintf("%q", e) == "" {
		t.Fatalf("expected non-empty simple formats")
	}
}

func TestEncodeNonConfigurationError(t *testing.T) {
	e := fmt.Errorf("plain")
	msg, details, payload := encodeConfigurationError(context.Background(), e)
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

func TestConfigurationFormatPercentV(t *testing.T) {
	e := ConfigurationErrorf("f", "iss", "m")
	s := fmt.Sprintf("%v", e)
	if s == "" {
		t.Fatalf("%%v produced empty string")
	}
	if !strings.Contains(s, "configuration error") {
		t.Fatalf("%%v missing expected text: %q", s)
	}
}
