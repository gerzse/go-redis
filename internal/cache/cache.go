package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/dgraph-io/ristretto"
	"log"

	"github.com/redis/go-redis/v9/internal"
)

// Cache makes it possible to store and retrieve keys and values from an in-memory cache.
// Backed by the ristretto library.
type Cache struct {
	cache *ristretto.Cache
}

type Config struct {
	MaxSize int64 // maximum size of the cache in bytes
	MaxKeys int64 // maximum number of keys to store in the cache
	// other configuration options:
	// - ttl (time to live) for cache entries
	// - eviction policy
}

// NewCache creates a new Cache instance with the given configuration
func NewCache(config *Config) (*Cache, error) {
	ristrettoConfig := &ristretto.Config{
		NumCounters: config.MaxKeys * 10, // number of keys to track frequency of (10x number of items to cache)
		MaxCost:     config.MaxSize,      // maximum cost of cache (in bytes)
		BufferItems: 64,                  // number of keys per Get buffer
		Metrics:     true,
	}

	cache, err := ristretto.NewCache(ristrettoConfig)
	if err != nil {
		return nil, err
	}

	return &Cache{cache: cache}, nil
}

func (c *Cache) Metrics() *ristretto.Metrics {
	return c.cache.Metrics
}

func (c *Cache) encodeKey(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(key); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *Cache) setKey(key, value interface{}) (bool, error) {
	encodedKey, err := c.encodeKey(key)
	if err != nil {
		return false, err
	}

	fmt.Println("Cache ", key, encodedKey, " and value ", value)
	set := c.cache.Set(encodedKey, value, 0)
	return set, nil
}

func (c *Cache) GetKey(key interface{}) (internal.Reader, bool, error) {
	encodedKey, err := c.encodeKey(key)
	if err != nil {
		return nil, false, err
	}
	fmt.Println("Get key", key, encodedKey)
	cachedData, b := c.cache.Get(encodedKey)
	if !b {
		return nil, false, nil
	}
	fmt.Println("Get key", key, encodedKey, cachedData)
	data, ok := cachedData.([]interface{})
	if !ok {
		log.Println("Wrong type returned from cache")
		return nil, false, nil
	}
	fmt.Println("Get key", key, encodedKey, data)

	replayer := &ReaderReplayer{
		data: data,
	}
	return replayer, b, nil
}

func (c *Cache) ClearKey(key interface{}) error {
	encodedKey, err := c.encodeKey(key)
	if err != nil {
		return err
	}
	c.cache.Del(encodedKey)
	return nil
}

func (c *Cache) Clear() {
	c.cache.Clear()
}

func (c *Cache) SpyReader(reader func(rd internal.Reader) error) (*ReaderSpy, func(rd internal.Reader) error) {
	spy := &ReaderSpy{
		cache: c,
		data:  make([]interface{}, 0),
	}
	wrappedReader := func(rd internal.Reader) error {
		return reader(spy.wrapReader(rd))
	}
	return spy, wrappedReader
}
