# Go-Cache: A High-Performance Multi-Layer Caching Library for Go

[![GoDoc](https://pkg.go.dev/badge/github.com/Jeanga7/go-cache-demo/pkg/cache)](https://pkg.go.dev/github.com/Jeanga7/go-cache-demo/pkg/cache)
[![Go Report Card](https://goreportcard.com/badge/github.com/jeanga7/go-cache-demo)](https://goreportcard.com/report/github.com/jeanga7/go-cache-demo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`go-cache` is a flexible and performant multi-layer caching library for Go applications. It combines a fast in-memory LRU cache for hot data with a distributed Redis backend for a larger, shared cache pool. This approach provides a robust solution to significantly reduce latency and decrease load on your primary data stores.

## Features

-   **Two-Layer Caching**: Fast in-memory LRU cache combined with a distributed Redis cache.
-   **Simple & Clean API**: Easy-to-use methods: `Get`, `Set`, `Purge`, and `Stats`.
-   **Configurable**: Easily configure TTL, memory cache size, and Redis connection details.
-   **Resilient**: Handles cache misses gracefully, allowing your application to fetch data from the source.
-   **Testable**: Designed to be easily testable with mocks (uses `redismock` in its own tests).
-   **Lightweight**: Minimal dependencies.

## Installation

To use this library in your project, simply use `go get`:

```bash
go get github.com/jeanga7/go-cache-demo/pkg/cache
```

## Quick Start

Here is a complete example of how to integrate the cache into your application.

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jeanga7/go-cache-demo/pkg/cache" // Adjust this import path to your project
)

// main demonstrates the core functionalities of the cache library.
func main() {
	// 1. Configure the cache
	opts := cache.Options{
		RedisAddr:     "localhost:6379", // Assumes Redis is running locally
		DefaultTTL:    2 * time.Second,      // Short TTL for demonstration
		MaxMemEntries: 100,
	}

	// 2. Initialize the cache
	myCache, err := cache.New(opts)
	if err != nil {
		log.Fatalf("FATAL: Could not connect to cache infrastructure: %v", err)
	}
	log.Println("✅ Cache library initialized successfully.")

	// 3. Use the cache
	userID := "user:profile:42"

	// First, try to get data from the cache
	log.Printf("\n[CLIENT] Attempting to get user '%s' profile...", userID)
	userData, err := myCache.Get(userID)

	// On a cache miss, fetch from the source and cache it
	if errors.Is(err, cache.ErrNotFound) {
		log.Println("   -> Cache MISS. Fetching from primary data source...")
		time.Sleep(50 * time.Millisecond) // Simulate slow DB call
		userData = fetchFromDatabase(userID)

		log.Printf("   -> Data fetched. Storing in cache (ID: %s).", userID)
		myCache.Set(userID, userData)
	} else if err != nil {
		log.Fatalf("An unexpected cache error occurred: %v", err)
	}

	log.Printf("   -> Cache HIT. User data: %s", string(userData))

	// Get it again - this time it will be a fast hit from memory
	log.Printf("\n[CLIENT] Requesting same user '%s' again...", userID)
	startTime := time.Now()
	userData, err = myCache.Get(userID)
	if err != nil {
		log.Fatalf("Should have been a cache hit, but got an error: %v", err)
	}
	log.Printf("   -> Cache HIT (from memory). Fetched in %s.", time.Since(startTime))

	// Purge the item from the cache
	log.Printf("\n[CLIENT] Purging user '%s' from cache...", userID)
	myCache.Purge(userID)
	_, err = myCache.Get(userID)
	log.Printf("   -> After purge, getting user '%s' results in: %v", userID, err)

	// Show some stats
	stats := myCache.Stats()
	fmt.Printf("\n--- Cache Stats ---\n")
	fmt.Printf("In-memory items: %d\n", stats.MemLen)
	fmt.Printf("--------------------\n")
}

// fetchFromDatabase simulates retrieving data from a slow backend.
func fetchFromDatabase(id string) []byte {
	return []byte(fmt.Sprintf(`{"id": "%s", "name": "Jean Cache"}`, id))
}
```

### Configuration Options

You can customize the cache behavior by passing an `Options` struct to the `New` function:

-   `RedisAddr` (string): Address of the Redis server (e.g., `"localhost:6379"`).
-   `DefaultTTL` (time.Duration): Default time-to-live for cache entries.
-   `MaxMemEntries` (int): The maximum number of entries to keep in the in-memory LRU cache.
-   `RedisClient` (*redis.Client): Allows you to provide an existing Redis client instance.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
