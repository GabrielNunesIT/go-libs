package cache

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/errors/errorspb"
)

func TestNewCacheError(t *testing.T) {
	err := CacheErrorf("get", "user:123", "connection timeout")
	if _, ok := err.(*cacheError); !ok {
		t.Fatalf("expected cacheError, got %T", err)
	}
}

func TestCacheErrorFormatting(t *testing.T) {
	err := CacheErrorf("set", "session:abc", "out of memory")
	errStr := err.Error()
	if !strings.Contains(errStr, "cache error") {
		t.Fatalf("error message missing 'cache error': %q", errStr)
	}
	if !strings.Contains(errStr, "set") {
		t.Fatalf("error message missing operation: %q", errStr)
	}
	if !strings.Contains(errStr, "session:abc") {
		t.Fatalf("error message missing key: %q", errStr)
	}
}

func TestCacheErrorSafeDetails(t *testing.T) {
	err := CacheErrorf("delete", "token:xyz", "permission denied")
	ce, ok := err.(*cacheError)
	if !ok {
		t.Fatalf("type assertion failed")
	}

	details := ce.SafeDetails()
	if len(details) != 3 {
		t.Fatalf("expected 3 safe details, got %d", len(details))
	}

	detailStr := strings.Join(details, "|")
	if !strings.Contains(detailStr, "operation: delete") {
		t.Fatalf("safe details missing operation: %v", details)
	}
	if !strings.Contains(detailStr, "key: token:xyz") {
		t.Fatalf("safe details missing key: %v", details)
	}
}

func TestEncodeDcodeCacheError(t *testing.T) {
	ce := &cacheError{operation: "get", key: "data:1", message: "network error"}

	msg, details, payload := encodeCacheError(context.Background(), ce)
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
	if sp.Msg != "network error" {
		t.Fatalf("payload msg = %q, want %q", sp.Msg, "network error")
	}

	// decode back
	got := decodeCacheError(context.Background(), msg, details, payload)
	ce2, ok := got.(*cacheError)
	if !ok {
		t.Fatalf("decoded error has wrong type: %T", got)
	}
	if ce2.operation != ce.operation || ce2.key != ce.key || ce2.message != ce.message {
		t.Fatalf("decoded value mismatch: %+v vs %+v", ce2, ce)
	}
}

func TestCacheErrorVerboseFormat(t *testing.T) {
	err := CacheErrorf("get", "key:123", "timeout")
	formatted := fmt.Sprintf("%+v", err)
	if formatted == "" {
		t.Fatalf("expected verbose formatting to produce output")
	}
	if !strings.Contains(formatted, "cacheError") {
		t.Fatalf("verbose format missing type name: %q", formatted)
	}
}

func TestCacheFormatPercentV(t *testing.T) {
	e := CacheErrorf("op", "k", "m")
	s := fmt.Sprintf("%v", e)
	if s == "" {
		t.Fatalf("%%v produced empty string")
	}
	if !strings.Contains(s, "cache error") {
		t.Fatalf("%%v missing expected text: %q", s)
	}
}

// Additional tests merged from cache_extra_test.go
// testPrinter implements errbase.Printer for unit tests.
type testPrinter struct {
	b      string
	detail bool
}

func (p *testPrinter) Print(args ...interface{})                 { p.b += fmt.Sprint(args...) }
func (p *testPrinter) Printf(format string, args ...interface{}) { p.b += fmt.Sprintf(format, args...) }
func (p *testPrinter) Detail() bool                              { return p.detail }

func TestCacheSafeFormatError(t *testing.T) {
	e := &cacheError{operation: "get", key: "k1", message: "m"}

	// call SafeFormatError directly
	p1 := &testPrinter{detail: false}
	next := e.SafeFormatError(p1)
	if next != nil {
		t.Fatalf("expected nil next, got %v", next)
	}

	p2 := &testPrinter{detail: true}
	next = e.SafeFormatError(p2)
	if next != nil {
		t.Fatalf("expected nil next, got %v", next)
	}

	// ensure Format branches for %s and %q are covered
	if fmt.Sprintf("%s", e) == "" {
		t.Fatalf("%%s produced empty string")
	}
	if fmt.Sprintf("%q", e) == "" {
		t.Fatalf("%%q produced empty string")
	}
}

func TestEncodeNonCacheError(t *testing.T) {
	// a plain error should be encoded as itself with no details/payload
	e := fmt.Errorf("plain")
	msg, details, payload := encodeCacheError(context.Background(), e)
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
