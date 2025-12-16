package cache

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Entry represents a cached file entry
type Entry struct {
	Content      string
	Size         int64
	ModifiedTime time.Time
	CachedAt     time.Time
	Hits         int64
}

// Cache is an LRU cache for file contents
type Cache struct {
	lru        *lru.Cache[string, *Entry]
	ttl        time.Duration
	mu         sync.RWMutex
	hits       int64
	misses     int64
	evictions  int64
}

// Stats represents cache statistics
type Stats struct {
	Size        int            `json:"size"`
	MaxSize     int            `json:"maxSize"`
	Hits        int64          `json:"hits"`
	Misses      int64          `json:"misses"`
	HitRate     float64        `json:"hitRate"`
	Evictions   int64          `json:"evictions"`
	TTL         string         `json:"ttl"`
	Entries     []EntryStats   `json:"entries,omitempty"`
}

// EntryStats represents stats for a single cache entry
type EntryStats struct {
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modifiedTime"`
	CachedAt     time.Time `json:"cachedAt"`
	Hits         int64     `json:"hits"`
}

// NewCache creates a new LRU cache
func NewCache(maxSize int, ttl time.Duration) (*Cache, error) {
	c := &Cache{
		ttl: ttl,
	}

	var err error
	c.lru, err = lru.NewWithEvict(maxSize, func(key string, value *Entry) {
		c.mu.Lock()
		c.evictions++
		c.mu.Unlock()
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Get retrieves an entry from the cache
func (c *Cache) Get(key string) (*Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.lru.Get(key)
	if !ok {
		c.misses++
		return nil, false
	}

	// Check TTL
	if time.Since(entry.CachedAt) > c.ttl {
		c.lru.Remove(key)
		c.misses++
		return nil, false
	}

	entry.Hits++
	c.hits++
	return entry, true
}

// Set adds an entry to the cache
func (c *Cache) Set(key string, entry *Entry) {
	entry.CachedAt = time.Now()
	c.lru.Add(key, entry)
}

// Remove removes an entry from the cache
func (c *Cache) Remove(key string) {
	c.lru.Remove(key)
}

// Clear clears the entire cache
func (c *Cache) Clear() {
	c.lru.Purge()
	c.mu.Lock()
	c.hits = 0
	c.misses = 0
	c.evictions = 0
	c.mu.Unlock()
}

// Stats returns cache statistics
func (c *Cache) Stats(detailed bool) Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	stats := Stats{
		Size:      c.lru.Len(),
		MaxSize:   c.lru.Len(), // LRU doesn't expose max size directly
		Hits:      c.hits,
		Misses:    c.misses,
		HitRate:   hitRate,
		Evictions: c.evictions,
		TTL:       c.ttl.String(),
	}

	if detailed {
		keys := c.lru.Keys()
		stats.Entries = make([]EntryStats, 0, len(keys))
		for _, key := range keys {
			if entry, ok := c.lru.Peek(key); ok {
				stats.Entries = append(stats.Entries, EntryStats{
					Path:         key,
					Size:         entry.Size,
					ModifiedTime: entry.ModifiedTime,
					CachedAt:     entry.CachedAt,
					Hits:         entry.Hits,
				})
			}
		}
	}

	return stats
}

// InvalidateStale removes entries older than their modification time
func (c *Cache) InvalidateStale(path string, modTime time.Time) bool {
	entry, ok := c.lru.Peek(path)
	if ok && entry.ModifiedTime.Before(modTime) {
		c.lru.Remove(path)
		return true
	}
	return false
}
