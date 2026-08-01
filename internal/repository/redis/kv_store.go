// Package redis provides a thin get/set/del-with-TTL helper over the shared
// redis.Client, used by session, cache, and stock-lock repositories added in
// later phases.
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type KVStore struct {
	client *redis.Client
}

func NewKVStore(client *redis.Client) *KVStore {
	return &KVStore{client: client}
}

func (s *KVStore) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *KVStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *KVStore) Del(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}
