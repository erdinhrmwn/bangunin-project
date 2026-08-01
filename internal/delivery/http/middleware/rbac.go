package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/response"
)

// RequireRole 403s unless the authenticated user's role is in roles (FR-2.8).
// Must run after Auth.
func RequireRole(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !slices.Contains(roles, ctxutil.Role(c)) {
			return c.Status(fiber.StatusForbidden).JSON(
				response.Error("Access denied", nil),
			)
		}
		return c.Next()
	}
}
