// Package route wires HTTP routes to handlers.
package route

import (
	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/delivery/http/handler"
)

// Register mounts all routes on app.
func Register(app *fiber.App, health *handler.HealthHandler) {
	app.Get("/health", health.Check)
}
