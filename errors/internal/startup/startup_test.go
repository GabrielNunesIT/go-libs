package startup

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/errors/errorspb"
)

func TestNewInitializationError(t *testing.T) {
	err := InitializationErrorf("database", "connection refused")
	if _, ok := err.(*initializationError); !ok {
		t.Fatalf("expected initializationError, got %T", err)
	}
}

func TestInitializationErrorFormatting(t *testing.T) {
	err := InitializationErrorf("cache", "failed to connect to Redis")
	errStr := err.Error()
	if !strings.Contains(errStr, "initialization error") {
		t.Fatalf("error message missing 'initialization error': %q", errStr)
	}
	if !strings.Contains(errStr, "cache") {
		t.Fatalf("error message missing component: %q", errStr)
	}
	if !strings.Contains(errStr, "Redis") {
		t.Fatalf("error message missing message content: %q", errStr)
	}
}

func TestInitializationErrorSafeDetails(t *testing.T) {
	err := InitializationErrorf("logger", "invalid log format configuration")
	ie, ok := err.(*initializationError)
	if !ok {
		t.Fatalf("type assertion failed")
	}

	details := ie.SafeDetails()
	if len(details) != 2 {
		t.Fatalf("expected 2 safe details, got %d", len(details))
	}

	detailStr := strings.Join(details, "|")
	if !strings.Contains(detailStr, "component: logger") {
		t.Fatalf("safe details missing component: %v", details)
	}
	if !strings.Contains(detailStr, "message:") {
		t.Fatalf("safe details missing message: %v", details)
	}
}

func TestEncodeDecodeInitializationError(t *testing.T) {
	ie := &initializationError{component: "config_loader", message: "config file not found"}

	msg, details, payload := encodeInitializationError(context.Background(), ie)
	if msg != ie.Error() {
		t.Fatalf("unexpected msg: %q, want %q", msg, ie.Error())
	}

	wantDetails := ie.SafeDetails()
	if len(details) != len(wantDetails) {
		t.Fatalf("unexpected safe details length: %d vs %d", len(details), len(wantDetails))
	}

	sp, ok := payload.(*errorspb.StringPayload)
	if !ok {
		t.Fatalf("payload type mismatch: %T", payload)
	}
	if sp.Msg != "config file not found" {
		t.Fatalf("payload msg = %q, want %q", sp.Msg, "config file not found")
	}

	// decode back
	got := decodeInitializationError(context.Background(), msg, details, payload)
	ie2, ok := got.(*initializationError)
	if !ok {
		t.Fatalf("decoded error has wrong type: %T", got)
	}
	if ie2.component != ie.component || ie2.message != ie.message {
		t.Fatalf("decoded value mismatch: %+v vs %+v", ie2, ie)
	}
}

func TestInitializationErrorVerboseFormat(t *testing.T) {
	err := InitializationErrorf("metrics", "Prometheus port already in use")
	formatted := fmt.Sprintf("%+v", err)
	if formatted == "" {
		t.Fatalf("expected verbose formatting to produce output")
	}
	if !strings.Contains(formatted, "initializationError") {
		t.Fatalf("verbose format missing type name: %q", formatted)
	}
}

// Additional tests merged from startup_extra_test.go
// testPrinter implements a minimal printer for SafeFormatError tests.
type testPrinter struct {
	b      string
	detail bool
}

func (p *testPrinter) Print(args ...interface{})                 { p.b += fmt.Sprint(args...) }
func (p *testPrinter) Printf(format string, args ...interface{}) { p.b += fmt.Sprintf(format, args...) }
func (p *testPrinter) Detail() bool                              { return p.detail }

func TestStartupSafeFormatError(t *testing.T) {
	e := &initializationError{component: "c", message: "m"}
	p := &testPrinter{detail: true}
	next := e.SafeFormatError(p)
	if next != nil {
		t.Fatalf("expected nil next, got %v", next)
	}

	if fmt.Sprintf("%s", e) == "" || fmt.Sprintf("%q", e) == "" {
		t.Fatalf("expected non-empty simple formats")
	}
}

func TestEncodeNonInitializationError(t *testing.T) {
	e := fmt.Errorf("plain")
	msg, details, payload := encodeInitializationError(context.Background(), e)
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

func TestInitializationFormatPercentV(t *testing.T) {
	e := InitializationErrorf("svc", "failed")
	s := fmt.Sprintf("%v", e)
	if s == "" {
		t.Fatalf("%%v produced empty string")
	}
	if !strings.Contains(s, "initialization error") {
		t.Fatalf("%%v missing expected text: %q", s)
	}
}
