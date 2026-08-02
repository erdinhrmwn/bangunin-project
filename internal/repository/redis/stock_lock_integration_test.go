//go:build integration

package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	goredis "github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"erdinhrmwn/bangunin/internal/repository/redis"
)

func setupRedis(t *testing.T) *goredis.Client {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(ctr) })

	uri, err := ctr.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := goredis.ParseURL(uri)
	require.NoError(t, err)

	return goredis.NewClient(opts)
}

func TestStockLock_AcquireRelease(t *testing.T) {
	client := setupRedis(t)
	lock := redis.NewStockLock(client)
	variantID := uuid.New()
	ctx := context.Background()

	ok, err := lock.Acquire(ctx, variantID, 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok, "first acquire should succeed")

	ok, err = lock.Acquire(ctx, variantID, 10*time.Second)
	require.NoError(t, err)
	require.False(t, ok, "second acquire while held should fail")

	require.NoError(t, lock.Release(ctx, variantID))

	ok, err = lock.Acquire(ctx, variantID, 10*time.Second)
	require.NoError(t, err)
	require.True(t, ok, "acquire after release should succeed")
}
