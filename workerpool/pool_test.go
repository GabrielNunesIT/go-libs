package workerpool_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GabrielNunesIT/go-libs/workerpool"
)

func TestPool_ProcessesAllTasks(t *testing.T) {
	t.Parallel()

	var count atomic.Int64

	pool := workerpool.New(context.Background(), func(_ context.Context, task int) {
		count.Add(int64(task))
	}, workerpool.WithWorkers[int](4))

	for i := 1; i <= 100; i++ {
		pool.Submit(i)
	}
	pool.Shutdown()

	expected := int64(5050) // sum 1..100
	if count.Load() != expected {
		t.Fatalf("expected sum %d, got %d", expected, count.Load())
	}
}

func TestPool_ConcurrentExecution(t *testing.T) {
	t.Parallel()

	var maxConcurrent atomic.Int64
	var current atomic.Int64

	pool := workerpool.New(context.Background(), func(_ context.Context, _ int) {
		cur := current.Add(1)
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		current.Add(-1)
	}, workerpool.WithWorkers[int](4), workerpool.WithBufferSize[int](100))

	for i := range 20 {
		pool.Submit(i)
	}
	pool.Shutdown()

	max := maxConcurrent.Load()
	if max < 2 || max > 4 {
		t.Fatalf("expected 2-4 concurrent workers, got %d", max)
	}
}

func TestPool_ShutdownWaitsForCompletion(t *testing.T) {
	t.Parallel()

	var completed atomic.Bool

	pool := workerpool.New(context.Background(), func(_ context.Context, _ int) {
		time.Sleep(50 * time.Millisecond)
		completed.Store(true)
	}, workerpool.WithWorkers[int](1))

	pool.Submit(1)
	pool.Shutdown()

	if !completed.Load() {
		t.Fatal("shutdown returned before task completed")
	}
}

func TestPool_ShutdownIdempotent(t *testing.T) {
	t.Parallel()

	pool := workerpool.New(context.Background(), func(_ context.Context, _ int) {
	}, workerpool.WithWorkers[int](2))

	pool.Submit(1)

	// Should not panic on double shutdown
	pool.Shutdown()
	pool.Shutdown()
}

func TestPool_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var processed atomic.Int64

	var mu sync.Mutex
	started := false

	pool := workerpool.New(ctx, func(_ context.Context, _ int) {
		mu.Lock()
		started = true
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		processed.Add(1)
	}, workerpool.WithWorkers[int](1), workerpool.WithBufferSize[int](100))

	// Submit a task, wait for it to start, then cancel
	pool.Submit(1)
	for {
		mu.Lock()
		s := started
		mu.Unlock()
		if s {
			break
		}
		time.Sleep(time.Millisecond)
	}

	// Submit more tasks that should be dropped on context cancel
	for i := range 10 {
		pool.Submit(i)
	}
	cancel()
	pool.Shutdown()

	// At least 1 should have processed, but not all 11
	got := processed.Load()
	if got == 0 {
		t.Fatal("expected at least 1 task processed")
	}
}

func TestPool_DefaultWorkerCount(t *testing.T) {
	t.Parallel()

	var count atomic.Int64

	// No WithWorkers — should default to runtime.NumCPU()
	pool := workerpool.New(context.Background(), func(_ context.Context, _ string) {
		count.Add(1)
	})

	pool.Submit("a")
	pool.Submit("b")
	pool.Shutdown()

	if count.Load() != 2 {
		t.Fatalf("expected 2, got %d", count.Load())
	}
}
