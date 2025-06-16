package cache

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/require"
)

func TestGetMiss(t *testing.T) {
	db, mock := redismock.NewClientMock()
	opts := Options{RedisClient: db, MaxMemEntries: 10, DefaultTTL: time.Minute}
	cache, err := New(opts)
	require.NoError(t, err)

	testID := "miss:key"

	// Expect a miss on both caches
	mock.ExpectGet(testID).RedisNil()

	_, err = cache.Get(testID)
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetAndGet(t *testing.T) {
	db, mock := redismock.NewClientMock()
	opts := Options{RedisClient: db, MaxMemEntries: 10, DefaultTTL: time.Minute}
	cache, err := New(opts)
	require.NoError(t, err)

	testID := "set:key"
	testData := []byte("some data")

	// Expect a set on redis
	mock.ExpectSet(testID, testData, cache.defaultTTL).SetVal("OK")

	cache.Set(testID, testData)

	// Now get it, should be in memory
	val, err := cache.Get(testID)
	require.NoError(t, err)
	require.Equal(t, testData, val)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCacheFlow(t *testing.T) {
	db, mock := redismock.NewClientMock()
	opts := Options{RedisClient: db, MaxMemEntries: 1, DefaultTTL: time.Minute}
	cache, err := New(opts)
	require.NoError(t, err)

	testID := "flow:key"
	testData := []byte("flow data")

	// 1. Initial Get -> Miss
	mock.ExpectGet(testID).RedisNil()
	_, err = cache.Get(testID)
	require.ErrorIs(t, err, ErrNotFound)

	// 2. Set
	mock.ExpectSet(testID, testData, cache.defaultTTL).SetVal("OK")
	cache.Set(testID, testData)

	// 3. Get -> Hit from memory
	val, err := cache.Get(testID)
	require.NoError(t, err)
	require.Equal(t, testData, val)

	// 4. Expire memory, Get -> Hit from Redis
	cache.memCache.Remove(testID)
	mock.ExpectGet(testID).SetVal(string(testData))
	val, err = cache.Get(testID)
	require.NoError(t, err)
	require.Equal(t, testData, val)

	// 5. Purge
	mock.ExpectDel(testID).SetVal(1)
	cache.Purge(testID)

	// 6. Get -> Miss
	mock.ExpectGet(testID).RedisNil()
	_, err = cache.Get(testID)
	require.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStats(t *testing.T) {
	db, _ := redismock.NewClientMock()
	opts := Options{RedisClient: db, MaxMemEntries: 10, DefaultTTL: time.Minute}
	cache, err := New(opts)
	require.NoError(t, err)

	cache.Set("key1", []byte("val1"))
	cache.Set("key2", []byte("val2"))

	stats := cache.Stats()
	require.Equal(t, 2, stats.MemLen)
	require.ElementsMatch(t, []interface{}{"key1", "key2"}, stats.MemKeys)
}
