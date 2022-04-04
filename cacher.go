package gohttpclient

import (
	"os"
	"path"
	"time"

	"github.com/go-redis/redis"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// ErrCacheKeyNotFound is a cached key does not exist error.
var ErrCacheKeyNotFound = errors.New("cache key not found")

// Cacher is the cached interface and requires Get and Set methods.
type Cacher interface {
	Get(key []byte) ([]byte, error)
	Set(key, value []byte, ttl time.Duration) error
}

// MemoryCache stores data in memory and implements the Cacher interface.
type MemoryCache struct {
	c *cache.Cache
}

// NewMemoryCache creates an in-memory cache instance.
// Note that the data it holds is limited by the operating system's memory resources.
func NewMemoryCache() MemoryCache {
	cleanupInterval := time.Second
	c := cache.New(cache.NoExpiration, cleanupInterval)
	return MemoryCache{c: c}
}

// Get gets the value of a key and returns ErrCacheKeyNotFound if it does not exist.
func (c MemoryCache) Get(key []byte) ([]byte, error) {
	value, found := c.c.Get(string(key))
	if !found {
		return nil, ErrCacheKeyNotFound
	}
	return value.([]byte), nil
}

// Set sets the value of the key, and configures the TTL of the cache.
func (c MemoryCache) Set(key, value []byte, ttl time.Duration) error {
	c.c.Set(string(key), value, ttl)
	return nil
}

// FileCache saves data to the file system and implements the Cacher interface.
type FileCache struct {
	RootDir     string
	TimeNowFunc func() time.Time
	Permission  os.FileMode
}

// NewFileCache creates an instance of the file system cache,
// and save the storage data in the rootDir directory in the form of files.
// Note that files are not removed periodically, only when they are accessed and found to be out of date.
func NewFileCache(rootDir string) FileCache {
	return FileCache{
		RootDir:     rootDir,
		TimeNowFunc: time.Now,
		Permission:  0644,
	}
}

func (c FileCache) path(key []byte) string {
	return path.Join(c.RootDir, string(key)+".cache")
}

// Get gets the value of a key and returns ErrCacheKeyNotFound if it does not exist.
func (c FileCache) Get(key []byte) ([]byte, error) {
	path := c.path(key)
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return nil, ErrCacheKeyNotFound
	} else if err != nil {
		return nil, errors.Wrapf(err, "Error checking if file exists, cache key '%s'", string(key))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading file contents, cache key '%s'", string(key))
	}

	var e fileCacheEntry
	err = msgpack.Unmarshal(data, &e)
	if err != nil {
		return nil, errors.Wrapf(err, "Error deserializing cached data, cache key '%s'", string(key))
	}

	nsec := e.TTL
	ttl := time.Unix(nsec/1e9, nsec%1e9)
	if ttl.Sub(c.TimeNowFunc()) >= 0 {
		return e.Value, nil
	}

	err = os.Remove(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error deleting an expired file, cache key '%s'", string(key))
	}

	return nil, ErrCacheKeyNotFound
}

// Set sets the value of the key, and configures the TTL of the cache.
func (c FileCache) Set(key, value []byte, ttl time.Duration) error {
	now := c.TimeNowFunc()
	e := fileCacheEntry{
		Key:   key,
		Value: value,
		Start: now.UnixNano(),
		TTL:   now.Add(ttl).UnixNano(),
	}

	data, err := msgpack.Marshal(&e)
	if err != nil {
		return errors.Wrapf(err, "Error serializing cached data, cache key '%s'", string(key))
	}
	path := c.path(key)
	err = os.WriteFile(path, data, c.Permission)
	return errors.Wrapf(err, "Error writing file contents, cache key '%s'", string(key))
}

type fileCacheEntry struct {
	Key   []byte
	Value []byte
	Start int64
	TTL   int64
}

// RedisCache stores data in redis server and implements the Cacher interface.
type RedisCache struct {
	c      *redis.Client
	Prefix string
}

// NewRedisCache creates an instance of the redis server cache,
// The default key has no prefix, of course you can set one yourself.
func NewRedisCache(c *redis.Client) RedisCache {
	return RedisCache{c: c, Prefix: ""}
}

func (c RedisCache) key(key []byte) string {
	return c.Prefix + string(key)
}

// Get gets the value of a key and returns ErrCacheKeyNotFound if it does not exist.
func (c RedisCache) Get(key []byte) ([]byte, error) {
	value, err := c.c.Get(c.key(key)).Result()
	if err == redis.Nil {
		return nil, ErrCacheKeyNotFound
	}
	if err != nil {
		return nil, errors.Wrapf(err, "Get for cache key '%s'", string(key))
	}
	return []byte(value), nil
}

// Set sets the value of the key, and configures the TTL of the cache.
func (c RedisCache) Set(key, value []byte, ttl time.Duration) error {
	_, err := c.c.Set(c.key(key), string(value), ttl).Result()
	return errors.Wrapf(err, "Set for cache key '%s'", string(key))
}
