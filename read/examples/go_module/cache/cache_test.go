package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheBasicSetAndGet(t *testing.T) {
	c := New(5 * time.Minute)

	err := c.Set("user:100", "Alice")
	if err != nil {
		t.Fatalf("Unexpected error on Set: %v", err)
	}

	val, err := c.Get("user:100")
	if err != nil {
		t.Fatalf("Unexpected error on Get: %v", err)
	}

	if val != "Alice" {
		t.Errorf("Expected 'Alice', got: %v", val)
	}

	stats := c.Stats()
	if stats.Hits != 1 || stats.Puts != 1 {
		t.Errorf("Stats mismatch: %+v", stats)
	}
}

func TestCacheEmptyKeyRejected(t *testing.T) {
	c := New(time.Minute)

	if err := c.Set("", "value"); err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey on Set, got: %v", err)
	}

	if _, err := c.Get(""); err != ErrInvalidKey {
		t.Errorf("Expected ErrInvalidKey on Get, got: %v", err)
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	c := New(10 * time.Millisecond)

	c.Set("ephemeral", 12345)

	val, err := c.Get("ephemeral")
	if err != nil || val != 12345 {
		t.Fatalf("Expected immediate hit, got: %v, err: %v", val, err)
	}

	// Wait past expiration
	time.Sleep(25 * time.Millisecond)

	_, err = c.Get("ephemeral")
	if err != ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound after expiration, got: %v", err)
	}

	stats := c.Stats()
	if stats.Evicts != 1 || stats.Misses != 1 {
		t.Errorf("Expected eviction and miss in stats: %+v", stats)
	}
}

func TestCachePurgeExpired(t *testing.T) {
	c := New(0) // No default TTL

	c.SetWithTTL("k1", "v1", 10*time.Millisecond)
	c.SetWithTTL("k2", "v2", 10*time.Millisecond)
	c.SetWithTTL("permanent", "v3", 0)

	if c.Len() != 3 {
		t.Fatalf("Expected 3 items, got: %d", c.Len())
	}

	time.Sleep(25 * time.Millisecond)

	evicted := c.PurgeExpired()
	if evicted != 2 {
		t.Errorf("Expected 2 evicted items, got: %d", evicted)
	}

	if c.Len() != 1 {
		t.Errorf("Expected 1 item remaining, got: %d", c.Len())
	}

	val, err := c.Get("permanent")
	if err != nil || val != "v3" {
		t.Errorf("Expected permanent item to persist, got: %v, err: %v", val, err)
	}
}

func TestCacheDeleteAndClear(t *testing.T) {
	c := New(time.Minute)

	c.Set("item1", "a")
	c.Set("item2", "b")

	if !c.Delete("item1") {
		t.Errorf("Expected Delete to return true for existing item")
	}

	if c.Delete("item1") {
		t.Errorf("Expected Delete to return false for already deleted item")
	}

	c.Clear()
	if c.Len() != 0 {
		t.Errorf("Expected 0 items after Clear, got: %d", c.Len())
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := New(time.Minute)
	const workers = 20
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(workers * 2)

	// Concurrent Writers
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key-%d-%d", workerID, j)
				_ = c.Set(key, j)
			}
		}(i)
	}

	// Concurrent Readers
	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("key-%d-%d", workerID, j)
				_, _ = c.Get(key)
			}
		}(i)
	}

	wg.Wait()

	if c.Len() != workers*iterations {
		t.Errorf("Expected %d items, got: %d", workers*iterations, c.Len())
	}
}
