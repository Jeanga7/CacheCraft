package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Jeanga7/go-cache-demo/pkg/cache"
)

// main now serves as a clean, runnable example of how to use the cache library.
func main() {
	// --- 1. Configuration ---
	// In a real application, these values would likely come from a config file.
	opts := cache.Options{
		RedisAddr:     "localhost:6379", // Assumes Redis is running locally
		DefaultTTL:    2 * time.Second,      // A short TTL for demonstration
		MaxMemEntries: 100,
	}

	// --- 2. Initialization ---
	myCache, err := cache.New(opts)
	if err != nil {
		// This would be a fatal error in a real app, as caching is critical.
		log.Fatalf("FATAL: Could not connect to cache infrastructure: %v", err)
	}
	log.Println("✅ Cache library initialized successfully.")

	// --- 3. Example Usage ---
	userID := "user:profile:42"

	// First, try to get the data from the cache.
	log.Printf("\n[CLIENT] Attempting to get user '%s' profile...", userID)
	userData, err := myCache.Get(userID)

	// If it's not in the cache (cache miss), we fetch it from the source.
	if errors.Is(err, cache.ErrNotFound) {
		log.Println("   -> Cache MISS. Fetching from primary data source (e.g., a database)...")
		// Simulate a slow database call
		time.Sleep(50 * time.Millisecond)
		userData = fetchFromDatabase(userID)

		// After fetching, we store it in the cache for next time.
		log.Printf("   -> Data fetched. Storing it in the cache now (ID: %s).", userID)
		myCache.Set(userID, userData)
	} else if err != nil {
		// Handle other potential errors (e.g., Redis connection issue)
		log.Fatalf("An unexpected cache error occurred: %v", err)
	}

	log.Printf("   -> Cache HIT. User data: %s", string(userData))

	// Get it again. This time it should be a fast cache hit.
	log.Printf("\n[CLIENT] Requesting same user '%s' profile again...", userID)
	startTime := time.Now()
	userData, err = myCache.Get(userID)
	if err != nil {
		log.Fatalf("Should have been a cache hit, but got an error: %v", err)
	}
	log.Printf("   -> Cache HIT (from memory). Fetched in %s.", time.Since(startTime))
	log.Printf("   -> User data: %s", string(userData))

	// Purge the cache
	log.Printf("\n[CLIENT] Purging user '%s' from cache...", userID)
	myCache.Purge(userID)
	_, err = myCache.Get(userID)
	log.Printf("   -> After purge, getting user '%s' results in: %v", userID, err)

	// Show stats
	stats := myCache.Stats()
	fmt.Printf("\n--- Cache Stats ---\n")
	fmt.Printf("In-memory items: %d\n", stats.MemLen)
	fmt.Printf("--------------------\n")
}

// fetchFromDatabase is a dummy function that simulates retrieving data from a slow
// backend, like a SQL database or another microservice.
func fetchFromDatabase(id string) []byte {
	// In a real app, you'd have your database query logic here.
	return []byte(fmt.Sprintf(`{"id": "%s", "name": "Jean Cache", "retrieved_at": "%s"}`, id, time.Now().UTC().Format(time.RFC3339)))
}
