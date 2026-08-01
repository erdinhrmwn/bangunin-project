package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/response"
)

// RequireApprovedSupplier 403s with SUPPLIER_NOT_APPROVED unless the
// authenticated user has a supplier profile with status=approved (FR-3.9).
// Must run after Auth + RequireRole(RoleSupplier).
func RequireApprovedSupplier(suppliers repository.SupplierRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, err := uuid.Parse(ctxutil.UserID(c))
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(
				response.Error("Access denied", fiber.Map{"code": "SUPPLIER_NOT_APPROVED"}),
			)
		}

		s, err := suppliers.FindByUserID(c.Context(), userID)
		if err != nil || s.Status != entity.SupplierStatusApproved {
			return c.Status(fiber.StatusForbidden).JSON(
				response.Error("Supplier account not approved", fiber.Map{"code": "SUPPLIER_NOT_APPROVED"}),
			)
		}
		return c.Next()
	}
}
