package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/redis/go-redis/v9"

	fiberredis "github.com/gofiber/storage/redis/v3"

	"erdinhrmwn/bangunin/pkg/response"
)

// AuthRateLimit is a stricter Redis-backed limiter for auth endpoints
// (FR-7.5): 5 req/min/IP, guarding against brute force and OTP abuse.
func AuthRateLimit(client *redis.Client) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:               5,
		Expiration:        time.Minute,
		Storage:           fiberredis.NewFromConnection(client),
		LimiterMiddleware: limiter.SlidingWindow{},
		// Prefixed so this limiter's Redis keys never collide with
		// RateLimit's — the underlying storage has no per-instance namespace.
		KeyGenerator: func(c fiber.Ctx) string { return "ratelimit:auth:" + c.IP() },
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(
				response.Error("Too many requests, please try again later", nil),
			)
		},
	})
}
