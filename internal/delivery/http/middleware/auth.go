package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/jwt"
	"erdinhrmwn/bangunin/pkg/response"
)

// Auth parses the Bearer JWT, rejects expired/invalid/blacklisted tokens
// (401), and injects claims into context via ctxutil (FR-2.7).
func Auth(jwtSvc *jwt.Service, authRepo repository.AuthRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			return unauthorized(c)
		}

		claims, err := jwtSvc.Parse(token)
		if err != nil {
			return unauthorized(c)
		}

		blacklisted, err := authRepo.IsJTIBlacklisted(c.Context(), claims.JTI)
		if err != nil || blacklisted {
			return unauthorized(c)
		}

		ctxutil.SetAuth(c, claims.UserID, claims.Role, claims.JTI)
		return c.Next()
	}
}

func unauthorized(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(
		response.Error("Authentication required", nil),
	)
}
