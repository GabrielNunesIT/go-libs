package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GabrielNunesIT/go-libs/retry"
)

var errTransient = errors.New("transient failure")

func TestDo_SucceedsFirstAttempt(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retry.Do(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesAndSucceeds(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retry.Do(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errTransient
		}
		return nil
	}, retry.WithMaxAttempts(5), retry.WithDelay(time.Millisecond), retry.WithJitter(false))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	t.Parallel()

	calls := 0
	err := retry.Do(context.Background(), func(_ context.Context) error {
		calls++
		return errTransient
	}, retry.WithMaxAttempts(3), retry.WithDelay(time.Millisecond), retry.WithJitter(false))

	if !errors.Is(err, errTransient) {
		t.Fatalf("expected errTransient, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	err := retry.Do(ctx, func(_ context.Context) error {
		calls++
		cancel() // Cancel after first attempt
		return errTransient
	}, retry.WithMaxAttempts(10), retry.WithDelay(time.Second))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestDo_Strategies(t *testing.T) {
	t.Parallel()

	strategies := []struct {
		name     string
		strategy retry.Strategy
	}{
		{"Constant", retry.StrategyConstant},
		{"Linear", retry.StrategyLinear},
		{"Exponential", retry.StrategyExponential},
	}

	for _, tt := range strategies {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			calls := 0
			err := retry.Do(context.Background(), func(_ context.Context) error {
				calls++
				if calls < 3 {
					return errTransient
				}
				return nil
			},
				retry.WithMaxAttempts(3),
				retry.WithStrategy(tt.strategy),
				retry.WithDelay(5*time.Millisecond),
				retry.WithJitter(false),
			)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			// Should have taken at least some time due to delays
			if time.Since(start) < 5*time.Millisecond {
				t.Fatalf("expected some delay, elapsed %v", time.Since(start))
			}
		})
	}
}

func TestDo_MaxDelayCap(t *testing.T) {
	t.Parallel()

	start := time.Now()
	calls := 0

	_ = retry.Do(context.Background(), func(_ context.Context) error {
		calls++
		return errTransient
	},
		retry.WithMaxAttempts(3),
		retry.WithStrategy(retry.StrategyExponential),
		retry.WithDelay(100*time.Millisecond),
		retry.WithMaxDelay(10*time.Millisecond), // Cap is smaller than base delay
		retry.WithJitter(false),
	)

	elapsed := time.Since(start)
	// With a 10ms cap and 2 sleeps, total should be ~20ms, not ~300ms
	if elapsed > 200*time.Millisecond {
		t.Fatalf("max delay cap not respected, elapsed %v", elapsed)
	}
}

func TestDo_DefaultOptions(t *testing.T) {
	t.Parallel()

	// Verify default config works (succeeds immediately)
	err := retry.Do(context.Background(), func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
