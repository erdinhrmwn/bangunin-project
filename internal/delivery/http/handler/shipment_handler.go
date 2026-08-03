package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/response"

	orderusecase "erdinhrmwn/bangunin/internal/usecase/order"
)

// ShipmentHandler serves read-only shipment lookup by order (FR-5.9),
// mounted under /user/orders/:id/shipment.
type ShipmentHandler struct {
	order *orderusecase.Usecase
}

func NewShipmentHandler(order *orderusecase.Usecase) *ShipmentHandler {
	return &ShipmentHandler{order: order}
}

func (h *ShipmentHandler) Get(c fiber.Ctx) error {
	orderID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	if _, err := h.order.GetForUser(c.Context(), orderID, userID); err != nil {
		return errJSON(c, err)
	}
	s, err := h.order.Shipment(c.Context(), orderID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", s))
}
