package gohttpclient

import (
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestFileCache(t *testing.T) {
	c := NewFileCache(os.TempDir())
	require.NotNil(t, c)

	key := []byte("c65fa2b3-4b8b-4485-af0e-3beea0d3373a")
	value := []byte("value")
	ttl := 100 * time.Millisecond

	err := c.Set(key, value, ttl)
	require.Nil(t, err)

	value2, err := c.Get(key)
	require.Nil(t, err)
	require.Equal(t, string(value), string(value2))

	time.Sleep(ttl)

	value2, err = c.Get(key)
	require.Equal(t, ErrCacheKeyNotFound, errors.Cause(err))
	require.Nil(t, value2)
}

func TestFileCache_WithError(t *testing.T) {
	c := NewFileCache(os.TempDir())
	require.NotNil(t, c)

	key := []byte("not_exists_key")
	_, err := c.Get(key)
	require.NotNil(t, err)
}

func TestRedisCache(t *testing.T) {
	c := NewRedisCache(getTestRedisClient())
	require.NotNil(t, c)

	key := []byte("c65fa2b3-4b8b-4485-af0e-3beea0d3373a")
	value := []byte("value")
	ttl := 100 * time.Millisecond

	err := c.Set(key, value, ttl)
	require.Nil(t, err)

	value2, err := c.Get(key)
	require.Nil(t, err)
	require.Equal(t, string(value), string(value2))

	time.Sleep(ttl)

	value2, err = c.Get(key)
	require.Equal(t, ErrCacheKeyNotFound, errors.Cause(err))
	require.Nil(t, value2)
}

func TestRedisCache_WithError(t *testing.T) {
	c := NewRedisCache(getTestRedisClient())
	require.NotNil(t, c)

	key := []byte("not_exists_key")
	_, err := c.Get(key)
	require.NotNil(t, err)

	rc := redis.NewClient(&redis.Options{
		Password: os.Getenv("REDIS_PASSWORD") + "_ERROR",
	})
	errClient := NewRedisCache(rc)
	_, err = errClient.Get(key)
	require.NotNil(t, err)
}

func getTestRedisClient() *redis.Client {
	c := redis.NewClient(&redis.Options{
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	return c
}
