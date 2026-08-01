package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/pkg/response"
)

const defaultBodyLimit = 2 << 20 // 2MB (FR-7.5); media upload is exempted and enforced by its own usecase-level limits.

// BodyLimit rejects request bodies over 2MB, except under /api/v1/media
// (image/PDF uploads use their own, larger limits in the media usecase).
func BodyLimit(c fiber.Ctx) error {
	if strings.HasPrefix(c.Path(), "/api/v1/media") {
		return c.Next()
	}
	if c.Request().Header.ContentLength() > defaultBodyLimit {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.Error("Request body too large", nil))
	}
	return c.Next()
}
