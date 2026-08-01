package middleware

import (
	"runtime/debug"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	"erdinhrmwn/bangunin/pkg/response"
)

// Recover catches panics, logs the stack, and responds with a 500 envelope
// instead of crashing the process.
func Recover(log zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Bytes("stack", debug.Stack()).
					Str("request_id", c.Get("X-Request-ID")).
					Msg("panic recovered")
				_ = c.Status(fiber.StatusInternalServerError).JSON(
					response.Error("Terjadi kesalahan pada server", nil),
				)
			}
		}()
		return c.Next()
	}
}
