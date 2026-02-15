// Package circuitbreaker implements a thread-safe circuit breaker pattern
// that guards calls to unreliable dependencies with three states:
// Closed (normal), Open (fail-fast), and Half-Open (probing).
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed allows all calls through. Failures are counted;
	// when the threshold is reached the circuit transitions to Open.
	StateClosed State = iota
	// StateOpen rejects all calls immediately with ErrCircuitOpen.
	// After the configured timeout the circuit transitions to Half-Open.
	StateOpen
	// StateHalfOpen allows a limited number of probe calls through.
	// On success the circuit resets to Closed; on failure it returns to Open.
	StateHalfOpen
)

const (
	defaultThreshold   = 5
	defaultTimeout     = 30 * time.Second
	defaultHalfOpenMax = 1
)

// ErrCircuitOpen is returned when a call is rejected because the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreaker guards calls to an unreliable dependency.
type CircuitBreaker struct {
	mu            sync.Mutex
	state         State
	failures      int
	successes     int // half-open probe successes
	lastFailure   time.Time
	threshold     int
	timeout       time.Duration
	halfOpenMax   int
	onStateChange func(from, to State)
	nowFunc       func() time.Time // injectable clock for testing
}

// Option configures the circuit breaker.
type Option func(*CircuitBreaker)

// WithThreshold sets the consecutive failure count that trips the circuit to Open.
// Default: 5.
func WithThreshold(n int) Option {
	return func(cb *CircuitBreaker) {
		if n > 0 {
			cb.threshold = n
		}
	}
}

// WithTimeout sets the duration the circuit stays Open before transitioning to Half-Open.
// Default: 30s.
func WithTimeout(d time.Duration) Option {
	return func(cb *CircuitBreaker) {
		cb.timeout = d
	}
}

// WithHalfOpenMax sets the maximum number of probe calls allowed in Half-Open state.
// If all probes succeed, the circuit resets to Closed.
// Default: 1.
func WithHalfOpenMax(n int) Option {
	return func(cb *CircuitBreaker) {
		if n > 0 {
			cb.halfOpenMax = n
		}
	}
}

// WithOnStateChange registers a callback invoked on every state transition.
func WithOnStateChange(fn func(from, to State)) Option {
	return func(cb *CircuitBreaker) {
		cb.onStateChange = fn
	}
}

// New creates a CircuitBreaker with the given options.
func New(opts ...Option) *CircuitBreaker {
	cb := &CircuitBreaker{
		state:       StateClosed,
		threshold:   defaultThreshold,
		timeout:     defaultTimeout,
		halfOpenMax: defaultHalfOpenMax,
		nowFunc:     time.Now,
	}

	for _, opt := range opts {
		opt(cb)
	}

	return cb
}

// Execute runs fn if the circuit allows it.
// Returns ErrCircuitOpen when the breaker is open and the timeout has not elapsed.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	// Evaluate current state, possibly transitioning Open → Half-Open.
	switch cb.state {
	case StateOpen:
		if cb.nowFunc().Sub(cb.lastFailure) >= cb.timeout {
			cb.transitionTo(StateHalfOpen)
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	case StateHalfOpen:
		// Already in half-open — allow if we haven't exceeded max probes.
		// Additional calls beyond halfOpenMax are rejected.
		if cb.successes >= cb.halfOpenMax {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	case StateClosed:
		// Allow through
	}

	cb.mu.Unlock()

	// Execute the function outside the lock.
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}

	return err
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forces the breaker back to Closed with zero counters.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	from := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0

	if from != StateClosed && cb.onStateChange != nil {
		cb.onStateChange(from, StateClosed)
	}
}

func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			cb.transitionTo(StateClosed)
		}
	case StateOpen:
		// Should not happen
	}
}

func (cb *CircuitBreaker) onFailure() {
	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.threshold {
			cb.lastFailure = cb.nowFunc()
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.lastFailure = cb.nowFunc()
		cb.transitionTo(StateOpen)
	case StateOpen:
		// Should not happen
	}
}

func (cb *CircuitBreaker) transitionTo(to State) {
	from := cb.state
	cb.state = to
	cb.failures = 0
	cb.successes = 0

	if cb.onStateChange != nil {
		cb.onStateChange(from, to)
	}
}
