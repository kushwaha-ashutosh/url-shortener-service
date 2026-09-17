// Package cache implements the cache-aside layer in front of Postgres
// for hot redirect lookups.
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrMiss = errors.New("cache miss")

const linkTTL = 24 * time.Hour

type Cache struct {
	rdb *redis.Client
}

func New(addr string) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{Addr: addr})}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}

func key(code string) string {
	return "link:" + code
}

// GetURL returns the cached long URL for a code, or ErrMiss if absent.
func (c *Cache) GetURL(ctx context.Context, code string) (string, error) {
	val, err := c.rdb.Get(ctx, key(code)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetURL populates the cache after a Postgres read (cache-aside fill).
func (c *Cache) SetURL(ctx context.Context, code, longURL string) error {
	return c.rdb.Set(ctx, key(code), longURL, linkTTL).Err()
}
