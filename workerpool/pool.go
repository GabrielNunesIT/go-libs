// Package workerpool provides a generic, bounded-concurrency worker pool with context support.
package workerpool

import (
	"context"
	"runtime"
	"sync"
)

// Pool is a generic worker pool that processes tasks of type T concurrently.
// Tasks are submitted via Submit and processed by a fixed number of worker goroutines.
type Pool[T any] struct {
	tasks   chan T
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	once    sync.Once
	handler func(ctx context.Context, task T)
}

// Option configures the pool.
type Option[T any] func(*poolConfig)

type poolConfig struct {
	workers    int
	bufferSize int
}

// WithWorkers sets the number of concurrent workers.
// Default: runtime.NumCPU().
func WithWorkers[T any](n int) Option[T] {
	return func(cfg *poolConfig) {
		if n > 0 {
			cfg.workers = n
		}
	}
}

// WithBufferSize sets the task channel buffer size.
// A larger buffer allows more tasks to be queued before Submit blocks.
// Default: equals worker count.
func WithBufferSize[T any](n int) Option[T] {
	return func(cfg *poolConfig) {
		if n > 0 {
			cfg.bufferSize = n
		}
	}
}

// New creates a Pool that runs handler for each submitted task.
// Workers are started immediately. The provided context controls the pool lifetime;
// when cancelled, workers will stop processing after their current task completes.
func New[T any](ctx context.Context, handler func(ctx context.Context, task T), opts ...Option[T]) *Pool[T] {
	cfg := &poolConfig{
		workers: runtime.NumCPU(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.bufferSize == 0 {
		cfg.bufferSize = cfg.workers
	}

	workerCtx, cancel := context.WithCancel(ctx)

	p := &Pool[T]{
		tasks:   make(chan T, cfg.bufferSize),
		cancel:  cancel,
		handler: handler,
	}

	p.wg.Add(cfg.workers)
	for range cfg.workers {
		go p.worker(workerCtx)
	}

	return p
}

// Submit enqueues a task for processing.
// Blocks if all workers are busy and the buffer is full (backpressure).
// Panics if called after Shutdown.
func (p *Pool[T]) Submit(task T) {
	p.tasks <- task
}

// Shutdown closes the task channel and waits for all in-flight tasks to complete.
// It is safe to call Shutdown multiple times; subsequent calls are no-ops.
func (p *Pool[T]) Shutdown() {
	p.once.Do(func() {
		close(p.tasks)
	})
	p.wg.Wait()
	p.cancel()
}

func (p *Pool[T]) worker(ctx context.Context) {
	defer p.wg.Done()
	for task := range p.tasks {
		select {
		case <-ctx.Done():
			return
		default:
			p.handler(ctx, task)
		}
	}
}
