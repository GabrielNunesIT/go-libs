package circuitbreaker_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GabrielNunesIT/go-libs/circuitbreaker"
)

var errDependency = errors.New("dependency failure")

func TestCircuitBreaker_ClosedAllowsCalls(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New()

	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_TripsOnThreshold(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(circuitbreaker.WithThreshold(3))

	for range 3 {
		_ = cb.Execute(func() error {
			return errDependency
		})
	}

	if cb.State() != circuitbreaker.StateOpen {
		t.Fatalf("expected StateOpen after 3 failures, got %v", cb.State())
	}

	// Additional calls should be rejected
	err := cb.Execute(func() error {
		return nil
	})
	if !errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(circuitbreaker.WithThreshold(3))

	// 2 failures, then a success
	_ = cb.Execute(func() error { return errDependency })
	_ = cb.Execute(func() error { return errDependency })
	_ = cb.Execute(func() error { return nil })

	// Should still be closed because success resets count
	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.State())
	}

	// 2 more failures shouldn't trip it
	_ = cb.Execute(func() error { return errDependency })
	_ = cb.Execute(func() error { return errDependency })

	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenOnTimeout(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(
		circuitbreaker.WithThreshold(1),
		circuitbreaker.WithTimeout(50*time.Millisecond),
	)

	// Trip the circuit
	_ = cb.Execute(func() error { return errDependency })

	if cb.State() != circuitbreaker.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Next call should transition to Half-Open and succeed
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("expected no error in half-open probe, got %v", err)
	}

	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed after successful probe, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReturnsToOpen(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(
		circuitbreaker.WithThreshold(1),
		circuitbreaker.WithTimeout(50*time.Millisecond),
	)

	// Trip the circuit
	_ = cb.Execute(func() error { return errDependency })

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Probe fails → back to Open
	err := cb.Execute(func() error { return errDependency })
	if err == nil {
		t.Fatal("expected error in half-open probe")
	}

	if cb.State() != circuitbreaker.StateOpen {
		t.Fatalf("expected StateOpen after failed probe, got %v", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(circuitbreaker.WithThreshold(1))

	_ = cb.Execute(func() error { return errDependency })

	if cb.State() != circuitbreaker.StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	cb.Reset()

	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed after reset, got %v", cb.State())
	}

	// Should allow calls again
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("expected no error after reset, got %v", err)
	}
}

func TestCircuitBreaker_OnStateChangeCallback(t *testing.T) {
	t.Parallel()

	var transitions []struct{ from, to circuitbreaker.State }

	cb := circuitbreaker.New(
		circuitbreaker.WithThreshold(2),
		circuitbreaker.WithTimeout(50*time.Millisecond),
		circuitbreaker.WithOnStateChange(func(from, to circuitbreaker.State) {
			transitions = append(transitions, struct{ from, to circuitbreaker.State }{from, to})
		}),
	)

	// Closed → Open
	_ = cb.Execute(func() error { return errDependency })
	_ = cb.Execute(func() error { return errDependency })

	time.Sleep(60 * time.Millisecond)

	// Open → Half-Open → Closed (via successful probe)
	_ = cb.Execute(func() error { return nil })

	if len(transitions) != 3 {
		t.Fatalf("expected 3 transitions, got %d: %+v", len(transitions), transitions)
	}

	// Closed → Open
	if transitions[0].from != circuitbreaker.StateClosed || transitions[0].to != circuitbreaker.StateOpen {
		t.Fatalf("expected Closed→Open, got %v→%v", transitions[0].from, transitions[0].to)
	}
	// Open → Half-Open
	if transitions[1].from != circuitbreaker.StateOpen || transitions[1].to != circuitbreaker.StateHalfOpen {
		t.Fatalf("expected Open→HalfOpen, got %v→%v", transitions[1].from, transitions[1].to)
	}
	// Half-Open → Closed
	if transitions[2].from != circuitbreaker.StateHalfOpen || transitions[2].to != circuitbreaker.StateClosed {
		t.Fatalf("expected HalfOpen→Closed, got %v→%v", transitions[2].from, transitions[2].to)
	}
}

func TestCircuitBreaker_HalfOpenMaxProbes(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(
		circuitbreaker.WithThreshold(1),
		circuitbreaker.WithTimeout(50*time.Millisecond),
		circuitbreaker.WithHalfOpenMax(3),
	)

	// Trip the circuit
	_ = cb.Execute(func() error { return errDependency })

	time.Sleep(60 * time.Millisecond)

	// Need 3 successful probes to close
	_ = cb.Execute(func() error { return nil })

	if cb.State() != circuitbreaker.StateHalfOpen {
		t.Fatalf("expected StateHalfOpen after 1/3 probes, got %v", cb.State())
	}

	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return nil })

	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed after 3/3 probes, got %v", cb.State())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New(circuitbreaker.WithThreshold(100))
	var executed atomic.Int64

	done := make(chan struct{})
	for range 50 {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 100 {
				_ = cb.Execute(func() error {
					executed.Add(1)
					return nil
				})
			}
		}()
	}

	for range 50 {
		<-done
	}

	if executed.Load() != 5000 {
		t.Fatalf("expected 5000 executions, got %d", executed.Load())
	}
}

func TestCircuitBreaker_DefaultOptions(t *testing.T) {
	t.Parallel()

	cb := circuitbreaker.New()

	// Default threshold is 5
	for range 4 {
		_ = cb.Execute(func() error { return errDependency })
	}
	if cb.State() != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed with 4 < 5 failures, got %v", cb.State())
	}

	_ = cb.Execute(func() error { return errDependency })
	if cb.State() != circuitbreaker.StateOpen {
		t.Fatalf("expected StateOpen at threshold 5, got %v", cb.State())
	}
}
