package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// StockLock provides a per-variant SETNX lock used by checkout confirm to
// serialize stock validation across concurrent requests (AC-5.b).
type StockLock struct {
	client *redis.Client
}

func NewStockLock(client *redis.Client) *StockLock {
	return &StockLock{client: client}
}

func lockKey(variantID uuid.UUID) string {
	return "stock_lock:" + variantID.String()
}

// Acquire attempts to lock variantID for ttl. Returns false if already held.
func (l *StockLock) Acquire(ctx context.Context, variantID uuid.UUID, ttl time.Duration) (bool, error) {
	return l.client.SetNX(ctx, lockKey(variantID), 1, ttl).Result()
}

func (l *StockLock) Release(ctx context.Context, variantID uuid.UUID) error {
	return l.client.Del(ctx, lockKey(variantID)).Err()
}
