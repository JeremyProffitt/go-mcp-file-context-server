package cache

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	if c == nil {
		t.Error("Expected non-nil cache")
	}
}

func TestCacheSetGet(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	entry := &Entry{
		Content:      "test content",
		Size:         12,
		ModifiedTime: time.Now(),
	}

	c.Set("/path/to/file", entry)

	retrieved, ok := c.Get("/path/to/file")
	if !ok {
		t.Error("Expected to find cached entry")
	}

	if retrieved.Content != "test content" {
		t.Errorf("Expected content 'test content', got %s", retrieved.Content)
	}
}

func TestCacheMiss(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	_, ok := c.Get("/nonexistent")
	if ok {
		t.Error("Expected cache miss for nonexistent key")
	}
}

func TestCacheTTL(t *testing.T) {
	// Create cache with very short TTL
	c, err := NewCache(100, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	entry := &Entry{
		Content:      "test",
		Size:         4,
		ModifiedTime: time.Now(),
	}

	c.Set("/path/to/file", entry)

	// Entry should exist immediately
	_, ok := c.Get("/path/to/file")
	if !ok {
		t.Error("Expected entry to exist immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(10 * time.Millisecond)

	// Entry should be expired now
	_, ok = c.Get("/path/to/file")
	if ok {
		t.Error("Expected entry to be expired after TTL")
	}
}

func TestCacheRemove(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	entry := &Entry{
		Content:      "test",
		Size:         4,
		ModifiedTime: time.Now(),
	}

	c.Set("/path/to/file", entry)

	// Verify entry exists
	_, ok := c.Get("/path/to/file")
	if !ok {
		t.Error("Expected entry to exist")
	}

	// Remove entry
	c.Remove("/path/to/file")

	// Verify entry is removed
	_, ok = c.Get("/path/to/file")
	if ok {
		t.Error("Expected entry to be removed")
	}
}

func TestCacheClear(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Add multiple entries
	for i := 0; i < 10; i++ {
		c.Set("/path/to/file"+string(rune('0'+i)), &Entry{
			Content:      "test",
			Size:         4,
			ModifiedTime: time.Now(),
		})
	}

	// Verify entries exist
	stats := c.Stats(false)
	if stats.Size != 10 {
		t.Errorf("Expected 10 entries, got %d", stats.Size)
	}

	// Clear cache
	c.Clear()

	// Verify cache is empty
	stats = c.Stats(false)
	if stats.Size != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.Size)
	}
}

func TestCacheStats(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Initial stats
	stats := c.Stats(false)
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Expected 0 hits and misses initially")
	}

	// Add an entry
	c.Set("/path/to/file", &Entry{
		Content:      "test",
		Size:         4,
		ModifiedTime: time.Now(),
	})

	// Cache hit
	c.Get("/path/to/file")
	stats = c.Stats(false)
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}

	// Cache miss
	c.Get("/nonexistent")
	stats = c.Stats(false)
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	// Check hit rate
	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", stats.HitRate)
	}
}

func TestCacheStatsDetailed(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	c.Set("/path/to/file1", &Entry{
		Content:      "content1",
		Size:         8,
		ModifiedTime: time.Now(),
	})

	c.Set("/path/to/file2", &Entry{
		Content:      "content2",
		Size:         8,
		ModifiedTime: time.Now(),
	})

	stats := c.Stats(true)
	if len(stats.Entries) != 2 {
		t.Errorf("Expected 2 detailed entries, got %d", len(stats.Entries))
	}
}

func TestCacheInvalidateStale(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	oldTime := time.Now().Add(-1 * time.Hour)
	c.Set("/path/to/file", &Entry{
		Content:      "old content",
		Size:         11,
		ModifiedTime: oldTime,
	})

	// Invalidate with newer mod time
	newTime := time.Now()
	invalidated := c.InvalidateStale("/path/to/file", newTime)
	if !invalidated {
		t.Error("Expected entry to be invalidated")
	}

	// Entry should be gone
	_, ok := c.Get("/path/to/file")
	if ok {
		t.Error("Expected entry to be removed after invalidation")
	}
}

func TestCacheHitCount(t *testing.T) {
	c, err := NewCache(100, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	c.Set("/path/to/file", &Entry{
		Content:      "test",
		Size:         4,
		ModifiedTime: time.Now(),
	})

	// Access multiple times
	for i := 0; i < 5; i++ {
		c.Get("/path/to/file")
	}

	entry, ok := c.Get("/path/to/file")
	if !ok {
		t.Fatal("Expected entry to exist")
	}

	// Should have 6 hits now (5 + 1 from the last get)
	if entry.Hits != 6 {
		t.Errorf("Expected 6 hits, got %d", entry.Hits)
	}
}
