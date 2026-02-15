// Package retry provides a generic retry executor with pluggable backoff strategies and context support.
package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// Strategy defines the backoff behavior between retry attempts.
type Strategy int

const (
	// StrategyConstant uses a fixed delay between attempts.
	StrategyConstant Strategy = iota
	// StrategyLinear increases the delay linearly (delay * attempt).
	StrategyLinear
	// StrategyExponential doubles the delay on each attempt.
	StrategyExponential
)

const (
	defaultMaxAttempts = 3
	defaultDelay       = 100 * time.Millisecond
	defaultMaxDelay    = 30 * time.Second
	jitterFraction     = 0.25
)

type config struct {
	maxAttempts int
	delay       time.Duration
	maxDelay    time.Duration
	strategy    Strategy
	jitter      bool
}

// Option configures the retry behavior.
type Option func(*config)

// WithMaxAttempts sets the maximum number of attempts (including the first call).
// Default: 3.
func WithMaxAttempts(n int) Option {
	return func(cfg *config) {
		if n > 0 {
			cfg.maxAttempts = n
		}
	}
}

// WithDelay sets the base delay between attempts.
// Default: 100ms.
func WithDelay(d time.Duration) Option {
	return func(cfg *config) {
		cfg.delay = d
	}
}

// WithMaxDelay caps the computed delay.
// Default: 30s.
func WithMaxDelay(d time.Duration) Option {
	return func(cfg *config) {
		cfg.maxDelay = d
	}
}

// WithStrategy sets the backoff strategy.
// Default: StrategyExponential.
func WithStrategy(s Strategy) Option {
	return func(cfg *config) {
		cfg.strategy = s
	}
}

// WithJitter enables or disables random ±25% jitter on the delay.
// Default: true.
func WithJitter(enabled bool) Option {
	return func(cfg *config) {
		cfg.jitter = enabled
	}
}

// Do executes fn, retrying on error according to the configured policy.
// It respects context cancellation between attempts.
// Returns the last error if all attempts fail or the context is cancelled.
func Do(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	cfg := &config{
		maxAttempts: defaultMaxAttempts,
		delay:       defaultDelay,
		maxDelay:    defaultMaxDelay,
		strategy:    StrategyExponential,
		jitter:      true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error
	for attempt := range cfg.maxAttempts {
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Don't sleep after the last attempt.
		if attempt == cfg.maxAttempts-1 {
			break
		}

		delay := computeDelay(cfg, attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// computeDelay calculates the backoff delay for the given attempt number.
func computeDelay(cfg *config, attempt int) time.Duration {
	var delay time.Duration

	switch cfg.strategy {
	case StrategyConstant:
		delay = cfg.delay
	case StrategyLinear:
		delay = cfg.delay * time.Duration(attempt+1)
	case StrategyExponential:
		//nolint:gosec // math.Pow on small ints is safe
		delay = cfg.delay * time.Duration(math.Pow(2, float64(attempt)))
	}

	if cfg.jitter {
		delta := float64(delay) * jitterFraction
		jitterVal := (rand.Float64()*2 - 1) * delta //nolint:gosec // jitter does not need crypto rand
		delay += time.Duration(jitterVal)
	}

	if delay > cfg.maxDelay {
		delay = cfg.maxDelay
	}
	if delay < 0 {
		delay = 0
	}

	return delay
}
