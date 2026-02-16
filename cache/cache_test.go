package cache_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GabrielNunesIT/go-libs/cache"
)

func TestCacheLRU(t *testing.T) {
	c := cache.New(cache.WithCapacity[string, int](2), cache.WithPolicy[string, int](cache.PolicyLRU))

	c.Set("a", 1)
	time.Sleep(time.Millisecond) // Ensure timestamps differ
	c.Set("b", 2)

	if c.Len() != 2 {
		t.Errorf("expected len 2, got %d", c.Len())
	}

	// Access "a" to make it most recently used
	val, ok := c.Get("a")
	if !ok || val != 1 {
		t.Errorf("expected to get 'a' = 1")
	}

	time.Sleep(time.Millisecond)
	// Add "c", should evict "b" (LRU) because "a" was just accessed
	c.Set("c", 3)

	if _, ok := c.Get("b"); ok {
		t.Errorf("expected 'b' to be evicted")
	}

	if _, ok := c.Get("a"); !ok {
		t.Errorf("expected 'a' to remain")
	}

	if _, ok := c.Get("c"); !ok {
		t.Errorf("expected 'c' to be present")
	}
}

func TestCacheFIFO(t *testing.T) {
	c := cache.New(cache.WithCapacity[string, int](2), cache.WithPolicy[string, int](cache.PolicyFIFO))

	c.Set("a", 1)
	time.Sleep(time.Millisecond)
	c.Set("b", 2)

	// Access "a", should NOT change eviction order in FIFO
	c.Get("a")

	time.Sleep(time.Millisecond)
	// Add "c", should evict "a" (First In)
	c.Set("c", 3)

	if _, ok := c.Get("a"); ok {
		t.Errorf("expected 'a' to be evicted")
	}

	if _, ok := c.Get("b"); !ok {
		t.Errorf("expected 'b' to remain")
	}

	if _, ok := c.Get("c"); !ok {
		t.Errorf("expected 'c' to be present")
	}
}

func TestCacheLFU(t *testing.T) {
	c := cache.New(cache.WithCapacity[string, int](2), cache.WithPolicy[string, int](cache.PolicyLFU))

	c.Set("a", 1)
	c.Set("b", 2)

	// Access "a" multiple times
	c.Get("a")
	c.Get("a")

	// Access "b" once
	c.Get("b")

	// Frequencies: a=3 (1 set + 2 get), b=2 (1 set + 1 get)
	// Add "c", should evict "b" (LFU)
	c.Set("c", 3)

	if _, ok := c.Get("b"); ok {
		t.Errorf("expected 'b' to be evicted")
	}

	if _, ok := c.Get("a"); !ok {
		t.Errorf("expected 'a' to remain")
	}

	if _, ok := c.Get("c"); !ok {
		t.Errorf("expected 'c' to be present")
	}
}

func TestCacheTTL(t *testing.T) {
	// Test expiration on Get
	c := cache.New(cache.WithTTL[string, int](50 * time.Millisecond))

	c.Set("a", 1)

	if _, ok := c.Get("a"); !ok {
		t.Errorf("expected 'a' to be present immediately")
	}

	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Get("a"); ok {
		t.Errorf("expected 'a' to be expired")
	}
}

func TestCachePolicyTTL(t *testing.T) {
	// Test eviction based on TTL
	c := cache.New(
		cache.WithCapacity[string, int](2),
		cache.WithPolicy[string, int](cache.PolicyTTL),
		cache.WithTTL[string, int](100*time.Millisecond),
	)

	c.Set("a", 1) // Expires in 100ms
	time.Sleep(10 * time.Millisecond)
	c.Set("b", 2) // Expires in 100ms from now (so later than a)

	// Add "c" with a very short TTL manually? No, WithTTL applies to all.
	// But we can simulate different expiration times by sleeping.

	// Current state:
	// "a" expires at T+100
	// "b" expires at T+110

	// Add "c" at T+20. Expires at T+120.
	// Capacity is 2. Must evict one.
	// PolicyTTL evicts the one expiring soonest.
	// "a" expires soonest.

	time.Sleep(10 * time.Millisecond)
	c.Set("c", 3)

	if _, ok := c.Get("a"); ok {
		t.Errorf("expected 'a' to be evicted (soonest expiration)")
	}

	if _, ok := c.Get("b"); !ok {
		t.Errorf("expected 'b' to remain")
	}

	if _, ok := c.Get("c"); !ok {
		t.Errorf("expected 'c' to remain")
	}
}

func TestCacheUpdate(t *testing.T) {
	c := cache.New(cache.WithCapacity[string, int](2))

	c.Set("a", 1)
	c.Set("a", 2)

	val, ok := c.Get("a")
	if !ok || val != 2 {
		t.Errorf("expected 'a' to be updated to 2")
	}

	if c.Len() != 1 {
		t.Errorf("expected len 1, got %d", c.Len())
	}
}

func TestCacheDelete(t *testing.T) {
	c := cache.New[string, int]()
	c.Set("a", 1)
	c.Delete("a")

	if c.Len() != 0 {
		t.Errorf("expected len 0, got %d", c.Len())
	}

	if _, ok := c.Get("a"); ok {
		t.Errorf("expected 'a' to be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	c := cache.New[string, int]()
	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("expected len 0, got %d", c.Len())
	}
}

func TestCachePolicyNone(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "SetAndGet",
			fn: func(t *testing.T) {
				c := cache.New(cache.WithPolicy[string, int](cache.PolicyNone))
				c.Set("a", 1)
				c.Set("b", 2)

				val, ok := c.Get("a")
				if !ok || val != 1 {
					t.Errorf("expected 'a' = 1, got %v, %v", val, ok)
				}
				val, ok = c.Get("b")
				if !ok || val != 2 {
					t.Errorf("expected 'b' = 2, got %v, %v", val, ok)
				}
			},
		},
		{
			name: "UnboundedGrowth",
			fn: func(t *testing.T) {
				c := cache.New(cache.WithPolicy[int, int](cache.PolicyNone))
				for i := range 1000 {
					c.Set(i, i)
				}
				if c.Len() != 1000 {
					t.Errorf("expected len 1000, got %d", c.Len())
				}
			},
		},
		{
			name: "Update",
			fn: func(t *testing.T) {
				c := cache.New(cache.WithPolicy[string, int](cache.PolicyNone))
				c.Set("a", 1)
				c.Set("a", 42)

				val, ok := c.Get("a")
				if !ok || val != 42 {
					t.Errorf("expected 'a' = 42, got %v", val)
				}
				if c.Len() != 1 {
					t.Errorf("expected len 1, got %d", c.Len())
				}
			},
		},
		{
			name: "Delete",
			fn: func(t *testing.T) {
				c := cache.New(cache.WithPolicy[string, int](cache.PolicyNone))
				c.Set("a", 1)
				c.Delete("a")

				if _, ok := c.Get("a"); ok {
					t.Errorf("expected 'a' to be deleted")
				}
				if c.Len() != 0 {
					t.Errorf("expected len 0, got %d", c.Len())
				}
			},
		},
		{
			name: "Clear",
			fn: func(t *testing.T) {
				c := cache.New(cache.WithPolicy[string, int](cache.PolicyNone))
				c.Set("a", 1)
				c.Set("b", 2)
				c.Clear()

				if c.Len() != 0 {
					t.Errorf("expected len 0, got %d", c.Len())
				}
			},
		},
		{
			name: "GetMiss",
			fn: func(t *testing.T) {
				c := cache.New(cache.WithPolicy[string, int](cache.PolicyNone))
				if _, ok := c.Get("missing"); ok {
					t.Errorf("expected miss for non-existent key")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestGetOrSet(t *testing.T) {
	tests := []struct {
		name      string
		fn        func(t *testing.T)
	}{
		{
			name: "CacheHit",
			fn: func(t *testing.T) {
				c := cache.New[string, int]()
				c.Set("a", 42)

				val, err := c.GetOrSet("a", func() (int, error) {
					t.Error("loader should not be called on cache hit")
					return 0, nil
				})
				if err != nil || val != 42 {
					t.Errorf("expected 42, got %d (err: %v)", val, err)
				}
			},
		},
		{
			name: "CacheMiss",
			fn: func(t *testing.T) {
				c := cache.New[string, int]()

				val, err := c.GetOrSet("a", func() (int, error) {
					return 99, nil
				})
				if err != nil || val != 99 {
					t.Errorf("expected 99, got %d (err: %v)", val, err)
				}

				// Verify it was cached.
				cached, ok := c.Get("a")
				if !ok || cached != 99 {
					t.Errorf("expected cached value 99, got %d", cached)
				}
			},
		},
		{
			name: "LoaderError",
			fn: func(t *testing.T) {
				c := cache.New[string, int]()
				errFail := errors.New("fail")

				_, err := c.GetOrSet("a", func() (int, error) {
					return 0, errFail
				})
				if !errors.Is(err, errFail) {
					t.Errorf("expected errFail, got %v", err)
				}

				// Verify error result was NOT cached.
				if _, ok := c.Get("a"); ok {
					t.Error("expected error result to not be cached")
				}
			},
		},
		{
			name: "Singleflight",
			fn: func(t *testing.T) {
				c := cache.New[string, int]()
				var calls atomic.Int32

				var wg sync.WaitGroup
				for range 100 {
					wg.Add(1)
					go func() {
						defer wg.Done()
						val, err := c.GetOrSet("key", func() (int, error) {
							calls.Add(1)
							time.Sleep(50 * time.Millisecond)
							return 7, nil
						})
						if err != nil || val != 7 {
							t.Errorf("expected 7, got %d (err: %v)", val, err)
						}
					}()
				}
				wg.Wait()

				if n := calls.Load(); n != 1 {
					t.Errorf("expected loader called exactly 1 time, got %d", n)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}
