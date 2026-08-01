// Package ctxutil stores/retrieves auth claims on fiber.Ctx locals, set by
// the Auth middleware and read by handlers/RequireRole (FR-2.7).
package ctxutil

import "github.com/gofiber/fiber/v3"

const (
	keyUserID = "auth.user_id"
	keyRole   = "auth.role"
	keyJTI    = "auth.jti"
)

func SetAuth(c fiber.Ctx, userID, role, jti string) {
	c.Locals(keyUserID, userID)
	c.Locals(keyRole, role)
	c.Locals(keyJTI, jti)
}

func UserID(c fiber.Ctx) string {
	v, _ := c.Locals(keyUserID).(string)
	return v
}

func Role(c fiber.Ctx) string {
	v, _ := c.Locals(keyRole).(string)
	return v
}

func JTI(c fiber.Ctx) string {
	v, _ := c.Locals(keyJTI).(string)
	return v
}
