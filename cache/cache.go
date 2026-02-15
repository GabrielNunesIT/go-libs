// Package cache provides a thread-safe generic cache with support for different eviction policies.
package cache

import (
	"container/heap"
	"container/list"
	"sync"
	"time"
)

// Policy defines the eviction policy for the cache.
type Policy int

const (
	// PolicyLRU (Least Recently Used) evicts the least recently accessed item.
	PolicyLRU Policy = iota
	// PolicyFIFO (First In, First Out) evicts the oldest item inserted.
	PolicyFIFO
	// PolicyLFU (Least Frequently Used) evicts the least frequently accessed item.
	PolicyLFU
	// PolicyTTL (Time To Live) evicts the item with the soonest expiration time.
	PolicyTTL
	// PolicyNone disables eviction entirely. The cache grows unboundedly;
	// entries are only removed via explicit Delete or Clear calls.
	// This is the most efficient policy when eviction is not needed.
	PolicyNone
)

// Cache is a thread-safe generic cache with support for different eviction policies.
type Cache[K comparable, V any] struct {
	mu        sync.RWMutex
	capacity  int
	policy    Policy
	ttl       time.Duration
	items     map[K]*entry[K, V]
	pq        *priorityQueue[K, V] // Used for LFU and TTL. Uses heap.
	evictList *list.List           // Used for LRU and FIFO. Doubly linked list.
}

type entry[K comparable, V any] struct {
	key           K
	value         V
	index         int           // heap index
	element       *list.Element // list element
	accessTime    int64         // UnixNano
	insertionTime int64         // UnixNano
	frequency     int
	expiration    int64 // UnixNano, 0 if no TTL
}

// priorityQueue implements heap.Interface
type priorityQueue[K comparable, V any] struct {
	items  []*entry[K, V]
	policy Policy
}

func (pq *priorityQueue[K, V]) Len() int { return len(pq.items) }

func (pq *priorityQueue[K, V]) Less(i, j int) bool {
	itemI := pq.items[i]
	itemJ := pq.items[j]

	switch pq.policy {
	case PolicyLFU:
		if itemI.frequency == itemJ.frequency {
			return itemI.accessTime < itemJ.accessTime // LRU tie-breaker
		}
		return itemI.frequency < itemJ.frequency
	case PolicyTTL:
		if itemI.expiration == 0 && itemJ.expiration == 0 {
			return itemI.accessTime < itemJ.accessTime
		}
		if itemI.expiration == 0 {
			return false // J expires sooner
		}
		if itemJ.expiration == 0 {
			return true // I expires sooner
		}
		return itemI.expiration < itemJ.expiration
	case PolicyLRU, PolicyFIFO:
		// Should not happen as these policies use evictList
		return itemI.accessTime < itemJ.accessTime
	default:
		// Fallback or other policies don't use PQ
		return itemI.accessTime < itemJ.accessTime
	}
}

func (pq *priorityQueue[K, V]) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

func (pq *priorityQueue[K, V]) Push(x any) {
	n := len(pq.items)
	//nolint:forcetypeassert // heap interface requires any
	item := x.(*entry[K, V])
	item.index = n
	pq.items = append(pq.items, item)
}

func (pq *priorityQueue[K, V]) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	pq.items = old[0 : n-1]
	return item
}

// Option defines a function to configure the cache.
type Option[K comparable, V any] func(*Cache[K, V])

// WithCapacity sets the maximum number of items in the cache.
// Default is 0 (unlimited).
func WithCapacity[K comparable, V any](capacity int) Option[K, V] {
	return func(cache *Cache[K, V]) {
		cache.capacity = capacity
	}
}

// WithPolicy sets the eviction policy.
// Default is PolicyLRU.
func WithPolicy[K comparable, V any](policy Policy) Option[K, V] {
	return func(cache *Cache[K, V]) {
		cache.policy = policy
	}
}

// WithTTL sets the default Time To Live for items.
// Default is 0 (no TTL).
func WithTTL[K comparable, V any](ttl time.Duration) Option[K, V] {
	return func(cache *Cache[K, V]) {
		cache.ttl = ttl
	}
}

// New creates a new Cache with the given options.
func New[K comparable, V any](opts ...Option[K, V]) *Cache[K, V] {
	cache := &Cache[K, V]{
		capacity: 0,
		policy:   PolicyLRU,
		items:    make(map[K]*entry[K, V]),
	}

	for _, opt := range opts {
		opt(cache)
	}

	switch cache.policy {
	case PolicyLRU, PolicyFIFO:
		cache.evictList = list.New()
	case PolicyLFU, PolicyTTL:
		cache.pq = &priorityQueue[K, V]{
			items:  make([]*entry[K, V], 0),
			policy: cache.policy,
		}
		heap.Init(cache.pq)
	case PolicyNone:
		// No eviction structures needed
	}

	return cache
}

// Set adds a value to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// PolicyNone: skip all metadata and eviction bookkeeping.
	if c.policy == PolicyNone {
		if item, ok := c.items[key]; ok {
			item.value = value
		} else {
			c.items[key] = &entry[K, V]{key: key, value: value}
		}
		return
	}

	now := time.Now().UnixNano()
	var expiration int64
	if c.ttl > 0 {
		expiration = now + int64(c.ttl)
	}

	// Check if item already exists
	if item, ok := c.items[key]; ok {
		// Update value
		item.value = value
		item.accessTime = now
		item.frequency++
		if c.ttl > 0 {
			item.expiration = expiration
		}

		switch c.policy {
		case PolicyLRU:
			c.evictList.MoveToFront(item.element)
		case PolicyLFU, PolicyTTL:
			heap.Fix(c.pq, item.index)
		case PolicyFIFO:
			// Do nothing
		}
		return
	}

	// Add new item
	if c.capacity > 0 && c.len() >= c.capacity {
		c.evict()
	}

	item := &entry[K, V]{
		key:           key,
		value:         value,
		accessTime:    now,
		insertionTime: now,
		frequency:     1,
		expiration:    expiration,
	}

	switch c.policy {
	case PolicyLRU, PolicyFIFO:
		elem := c.evictList.PushFront(item)
		item.element = elem
	case PolicyLFU, PolicyTTL:
		heap.Push(c.pq, item)
	}
	c.items[key] = item
}

// Get retrieves a value from the cache.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	// PolicyNone: read-only map lookup under RLock.
	if c.policy == PolicyNone {
		c.mu.RLock()
		defer c.mu.RUnlock()

		if item, ok := c.items[key]; ok {
			return item.value, true
		}
		var zero V
		return zero, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		// Check TTL
		if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
			c.removeElement(item)
			var zero V
			return zero, false
		}

		item.accessTime = time.Now().UnixNano()
		item.frequency++

		switch c.policy {
		case PolicyLRU:
			c.evictList.MoveToFront(item.element)
		case PolicyLFU, PolicyTTL:
			heap.Fix(c.pq, item.index)
		case PolicyFIFO:
			// Do nothing
		}
		return item.value, true
	}

	var zero V
	return zero, false
}

// Delete removes a key from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		c.removeElement(item)
	}
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// len returns the number of items without acquiring the lock.
// Must be called while holding c.mu.
func (c *Cache[K, V]) len() int {
	return len(c.items)
}

// Clear removes all items from the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[K]*entry[K, V])
	if c.evictList != nil {
		c.evictList.Init()
	}
	if c.pq != nil {
		c.pq.items = make([]*entry[K, V], 0)
		heap.Init(c.pq)
	}
}

// evict removes the item based on policy.
func (c *Cache[K, V]) evict() {
	switch c.policy {
	case PolicyLRU, PolicyFIFO:
		elem := c.evictList.Back()
		if elem != nil {
			//nolint:forcetypeassert // evictList contains *entry[K, V]
			c.removeElement(elem.Value.(*entry[K, V]))
		}
	case PolicyLFU, PolicyTTL:
		if c.pq.Len() > 0 {
			//nolint:forcetypeassert // pq contains *entry[K, V]
			item := heap.Pop(c.pq).(*entry[K, V])
			delete(c.items, item.key)
		}
	case PolicyNone:
		// No eviction
	}
}

func (c *Cache[K, V]) removeElement(item *entry[K, V]) {
	switch c.policy {
	case PolicyLRU, PolicyFIFO:
		c.evictList.Remove(item.element)
	case PolicyLFU, PolicyTTL:
		heap.Remove(c.pq, item.index)
	case PolicyNone:
		// No eviction structures to clean up
	}
	delete(c.items, item.key)
}
