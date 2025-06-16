// Package cache provides a high-performance, multi-layer caching library for Go.
// It combines an in-memory LRU cache for hot data with a Redis backend for a larger,
// distributed cache, offering a robust solution for reducing latency and database load.
package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a requested item is not found in any cache layer.
var ErrNotFound = errors.New("item not found in cache")

// cacheEntry is an internal struct representing a single item in the in-memory cache.
type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// Options holds the configuration for creating a new Cache instance.
// It allows for customization of connection details, cache sizes, and expiration policies.
type Options struct {
	// RedisAddr is the address of the Redis server (e.g., "localhost:6379").
	// This is ignored if RedisClient is provided.
	RedisAddr string
	// DefaultTTL is the default time-to-live for cache entries.
	DefaultTTL time.Duration
	// MaxMemEntries is the maximum number of entries to keep in the in-memory LRU cache.
	MaxMemEntries int
	// RedisClient allows providing an existing Redis client. If nil, a new client is created.
	RedisClient *redis.Client
}

// Cache is the main cache controller. It orchestrates the flow of data
// between the in-memory LRU cache and the Redis cache.
type Cache struct {
	memCache    *lru.Cache
	redisClient *redis.Client
	defaultTTL  time.Duration
	ctx         context.Context
}

// Stats contains statistics about the cache's performance.
type Stats struct {
	MemHits   int
	MemMisses int
	MemEvicts int
	MemLen    int
	MemKeys   []interface{}
}

// New initializes a new multi-layer Cache with the given options.
// It returns an error if the configuration is invalid or if the connection to Redis fails.
func New(opts Options) (*Cache, error) {
	memCache, err := lru.New(opts.MaxMemEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory cache: %w", err)
	}

	redisClient := opts.RedisClient
	ctx := context.Background()

	if redisClient == nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr: opts.RedisAddr,
			DB:   0, // use default DB
		})

		if _, err := redisClient.Ping(ctx).Result(); err != nil {
			return nil, fmt.Errorf("failed to connect to redis: %w", err)
		}
	}

	return &Cache{
		memCache:    memCache,
		redisClient: redisClient,
		defaultTTL:  opts.DefaultTTL,
		ctx:         ctx,
	}, nil
}

// Get retrieves an item from the cache. It checks the in-memory LRU cache first,
// then the Redis cache. If the item is not found in either, it returns ErrNotFound.
func (c *Cache) Get(id string) ([]byte, error) {
	// 1. Check in-memory cache
	if entryRaw, ok := c.memCache.Get(id); ok {
		entry := entryRaw.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.value, nil
		}
		c.memCache.Remove(id) // Expired
	}

	// 2. Check Redis
	val, err := c.redisClient.Get(c.ctx, id).Bytes()
	if err == nil {
		// Populate in-memory cache for subsequent fast access
		c.memCache.Add(id, cacheEntry{
			value:     val,
			expiresAt: time.Now().Add(c.defaultTTL),
		})
		return val, nil
	}
	if err != redis.Nil {
		return nil, fmt.Errorf("redis GET error: %w", err)
	}

	// 3. Not found in any cache
	return nil, ErrNotFound
}

// Set stores an item in both cache layers with the default TTL.
func (c *Cache) Set(id string, value []byte) {
	expiresAt := time.Now().Add(c.defaultTTL)
	// Store in memory
	c.memCache.Add(id, cacheEntry{value: value, expiresAt: expiresAt})
	// Store in Redis
	c.redisClient.Set(c.ctx, id, value, c.defaultTTL)
}

// Purge removes an item from both cache layers.
func (c *Cache) Purge(id string) {
	c.memCache.Remove(id)
	c.redisClient.Del(c.ctx, id)
}

// Stats returns statistics for the in-memory cache.
func (c *Cache) Stats() Stats {
	// Note: lru.Cache is not safe for concurrent access to its stats fields.
	// In a real-world high-concurrency scenario, this might require locking.
	return Stats{
		MemLen:  c.memCache.Len(),
		MemKeys: c.memCache.Keys(),
	}
}
