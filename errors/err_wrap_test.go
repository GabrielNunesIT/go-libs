package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GabrielNunesIT/go-libs/errors"
)

func TestWrapAndAnnotations(t *testing.T) {
	base := errors.New("base")
	w := errors.Wrap(base, "pref")
	if w == nil || !strings.Contains(w.Error(), "pref") || !strings.Contains(w.Error(), "base") {
		t.Fatalf("Wrap produced unexpected error: %v", w)
	}

	wf := errors.Wrapf(base, "pfx-%s", "v")
	if wf == nil || !strings.Contains(wf.Error(), "pfx-v") {
		t.Fatalf("Wrapf produced unexpected error: %v", wf)
	}

	// empty format: should still return a wrapped error (no prefix added)
	wf2 := errors.Wrapf(base, "", "ignored")
	if wf2 == nil || !strings.Contains(wf2.Error(), "base") {
		t.Fatalf("Wrapf(empty) produced unexpected error: %v", wf2)
	}

	// WithStack/WithStackDepth should preserve the original message
	ws := errors.WithStack(base)
	if ws == nil || !strings.Contains(ws.Error(), "base") {
		t.Fatalf("WithStack produced unexpected error: %v", ws)
	}

	ws2 := errors.WithStackDepth(base, 2)
	if ws2 == nil || !strings.Contains(ws2.Error(), "base") {
		t.Fatalf("WithStackDepth produced unexpected error: %v", ws2)
	}

	// WithDetail and WithHint embed extra information visible with %+v
	det := errors.WithDetail(base, "my-detail")
	if det == nil {
		t.Fatalf("WithDetail returned nil")
	}
	if !strings.Contains(fmt.Sprintf("%+v", det), "my-detail") {
		t.Fatalf("WithDetail did not include detail when formatted: %+v", det)
	}

	hint := errors.WithHint(base, "try this")
	if hint == nil {
		t.Fatalf("WithHint returned nil")
	}
	if !strings.Contains(fmt.Sprintf("%+v", hint), "try this") {
		t.Fatalf("WithHint did not include hint when formatted: %+v", hint)
	}

	// WithDomain should produce a non-nil error
	dom := errors.WithDomain(base, "example.com/mypkg")
	if dom == nil {
		t.Fatalf("WithDomain returned nil")
	}
}
