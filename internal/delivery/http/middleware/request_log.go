package middleware

import (
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// fourXXSampleRate logs 1 in N 4xx responses (FR-7.6) — 5xx and 2xx/3xx are
// always logged; 4xx (mostly client validation noise) is sampled to cut volume.
const fourXXSampleRate = 10

var fourXXCounter uint64

// RequestLog logs method, path, status, latency, and request_id per request.
// 4xx responses are sampled at 1/fourXXSampleRate to reduce log volume.
func RequestLog(log zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		status := c.Response().StatusCode()
		if status >= 400 && status < 500 {
			if atomic.AddUint64(&fourXXCounter, 1)%fourXXSampleRate != 0 {
				return err
			}
		}

		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Dur("latency", time.Since(start)).
			Str("request_id", c.Get("X-Request-ID")).
			Msg("request")

		return err
	}
}
