package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/redis/go-redis/v9"

	fiberredis "github.com/gofiber/storage/redis/v3"

	"erdinhrmwn/bangunin/pkg/response"
)

// RateLimit is a Redis-backed sliding-window limiter: 60 req/min/IP by
// default, skipping /health and /metrics so uptime checks and Prometheus
// scrapes are never throttled.
func RateLimit(client *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               60,
		Expiration:        time.Minute,
		Storage:           fiberredis.NewFromConnection(client),
		LimiterMiddleware: limiter.SlidingWindow{},
		Next:              func(c fiber.Ctx) bool { return c.Path() == "/health" || c.Path() == "/metrics" },
		// Prefixed so this limiter's Redis keys never collide with
		// AuthRateLimit's — the underlying storage has no per-instance namespace.
		KeyGenerator: func(c fiber.Ctx) string { return "ratelimit:global:" + c.IP() },
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(
				response.Error("Too many requests, please try again later", nil),
			)
		},
	})
}
