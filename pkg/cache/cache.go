// Package cache provides a small generic JSON cache-aside helper on top of
// the shared redis.Client (category tree, product detail, etc.).
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetJSON returns the cached value and true, or the zero value and false on
// a cache miss. Any other redis/unmarshal error is returned as err.
func GetJSON[T any](ctx context.Context, rdb *redis.Client, key string) (T, bool, error) {
	var v T
	raw, err := rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return v, false, nil
	}
	if err != nil {
		return v, false, fmt.Errorf("cache: get %s: %w", key, err)
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, false, fmt.Errorf("cache: unmarshal %s: %w", key, err)
	}
	return v, true, nil
}

func SetJSON(ctx context.Context, rdb *redis.Client, key string, v any, ttl time.Duration) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("cache: marshal %s: %w", key, err)
	}
	if err := rdb.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %s: %w", key, err)
	}
	return nil
}

// Delete removes a cache key; a missing key is not an error.
func Delete(ctx context.Context, rdb *redis.Client, key string) error {
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("cache: del %s: %w", key, err)
	}
	return nil
}

// ProductDetailKey is the cache key for a product detail lookup, shared
// between the catalog usecase (reader) and product usecase (invalidator on
// mutation) so neither has to import the other.
func ProductDetailKey(slug string) string {
	return "catalog:product:" + slug
}
