package cache_test

import (
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
