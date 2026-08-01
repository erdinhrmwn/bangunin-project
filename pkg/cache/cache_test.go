package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/pkg/cache"
)

func TestGetSetJSON(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	type payload struct {
		Name string `json:"name"`
	}
	key := "cache_test:payload"
	defer func() { _ = rdb.Del(ctx, key).Err() }()

	_, ok, err := cache.GetJSON[payload](ctx, rdb, key)
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, cache.SetJSON(ctx, rdb, key, payload{Name: "semen"}, time.Minute))

	got, ok, err := cache.GetJSON[payload](ctx, rdb, key)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "semen", got.Name)

	require.NoError(t, cache.Delete(ctx, rdb, key))
	_, ok, err = cache.GetJSON[payload](ctx, rdb, key)
	require.NoError(t, err)
	require.False(t, ok)
}
