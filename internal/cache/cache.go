package cache

import (
	"time"

	"github.com/DmitriyVTitov/size"
	"github.com/dgraph-io/ristretto"
)

// Cache is a wrapper around the `ristretto` cache that makes namespaces explicit.
type Cache struct {
	cache *ristretto.Cache
}

// Stats are the cache stats.
type Stats struct {
	Hits          uint64 // Total number of hits.
	Misses        uint64 // Total number of misses.
	ItemCount     uint64 // Current item count.
	TotalItemSize uint64 // Current sum of all item sizes.
}

// New returns a new cache.
func New(sizeInBytes int64) (*Cache, error) {
	c := &Cache{}

	var err error
	c.cache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 10 * 5000,                                                      // Number of keys to track frequency of; this should be 10x the number of expected keys to keep (5,000).
		MaxCost:     sizeInBytes,                                                    // This is the maximum cost of all of the items in the cache (this will be their size).
		BufferItems: 64,                                                             // Number of keys per Get buffer; this should be 64 according to the docs.
		Metrics:     true,                                                           // Track metrics.
		Cost:        func(input interface{}) int64 { return int64(size.Of(input)) }, // Use the size of the data as the cost.
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// makeKey creates a key with the appropriate namespace.
func (c *Cache) makeKey(namespace, key string) string {
	return namespace + "::" + key
}

// Get an entry from the cache.
func (c *Cache) Get(namespace, key string) (value interface{}, found bool) {
	return c.cache.Get(c.makeKey(namespace, key))
}

// Set an entry in the cache.
func (c *Cache) Set(namespace, key string, value interface{}) {
	cost := int64(0) // Use zero to have it automatically compute the cost.
	c.cache.Set(c.makeKey(namespace, key), value, cost)
}

// SetWithTTL an entry in the cache with a TTL.
func (c *Cache) SetWithTTL(namespace, key string, value interface{}, ttl time.Duration) {
	cost := int64(0) // Use zero to have it automatically compute the cost.
	c.cache.SetWithTTL(c.makeKey(namespace, key), value, cost, ttl)
}

// Del delete an entry from the cache.
func (c *Cache) Del(namespace, key string) {
	c.cache.Del(c.makeKey(namespace, key))
}

// Stats returns the cache stats.
func (c *Cache) Stats() Stats {
	return Stats{
		Hits:          c.cache.Metrics.Hits(),
		Misses:        c.cache.Metrics.Misses(),
		ItemCount:     c.cache.Metrics.KeysAdded() - c.cache.Metrics.KeysEvicted(),
		TotalItemSize: c.cache.Metrics.CostAdded() - c.cache.Metrics.CostEvicted(),
	}
}
