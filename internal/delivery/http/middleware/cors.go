package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORS returns a configurable CORS handler; empty allowedOrigins means "*".
func CORS(allowedOrigins []string) fiber.Handler {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	})
}
