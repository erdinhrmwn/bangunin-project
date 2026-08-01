package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// RequestID reads X-Request-ID, generating one if absent, and echoes it back
// on the response.
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
			c.Request().Header.Set("X-Request-ID", id)
		}
		c.Set("X-Request-ID", id)
		return c.Next()
	}
}
