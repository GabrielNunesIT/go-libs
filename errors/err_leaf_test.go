package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GabrielNunesIT/go-libs/errors"
	"github.com/GabrielNunesIT/go-libs/errors/internal/assert"
)

func TestLeafHelpers(t *testing.T) {
	e := errors.New("hello")
	if e == nil || !strings.Contains(e.Error(), "hello") {
		t.Fatalf("New produced unexpected error: %v", e)
	}

	ef := errors.Newf("%s-%d", "x", 1)
	if ef == nil || !strings.Contains(ef.Error(), "x-1") {
		t.Fatalf("Newf produced unexpected error: %v", ef)
	}

	af := errors.AssertionFailedf("oops: %s", "bad")
	if !assert.IsAssertionFailure(af) {
		t.Fatalf("AssertionFailedf not recognized as assertion failure")
	}
}

func TestPublicWrappers(t *testing.T) {
	ce := errors.CacheErrorf("get", "k", "m")
	if ce == nil || !strings.Contains(ce.Error(), "cache error") {
		t.Fatalf("CacheErrorf produced unexpected error: %v", ce)
	}

	ie := errors.InitializationErrorf("svc", "failed")
	if ie == nil || !strings.Contains(ie.Error(), "initialization error") {
		t.Fatalf("InitializationErrorf produced unexpected error: %v", ie)
	}

	cfg := errors.ConfigurationErrorf("f", "missing", "msg")
	if cfg == nil || !strings.Contains(cfg.Error(), "configuration error") {
		t.Fatalf("ConfigurationErrorf produced unexpected error: %v", cfg)
	}

	// Ensure verbose formatting shows internal type names
	if !strings.Contains(fmt.Sprintf("%+v", ce), "cacheError") {
		t.Fatalf("verbose formatting missing cacheError type")
	}
	if !strings.Contains(fmt.Sprintf("%+v", ie), "initializationError") {
		t.Fatalf("verbose formatting missing initializationError type")
	}
	if !strings.Contains(fmt.Sprintf("%+v", cfg), "configurationError") {
		t.Fatalf("verbose formatting missing configurationError type")
	}
}

func TestValidationWrapper(t *testing.T) {
	v := errors.ValidationFailedf("email", "bad", "required")
	if v == nil || !strings.Contains(v.Error(), "validation error") {
		t.Fatalf("ValidationFailedf produced unexpected error: %v", v)
	}
}
