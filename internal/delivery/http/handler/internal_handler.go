package handler

import (
	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/usecase/order"
	"erdinhrmwn/bangunin/pkg/response"
)

// InternalHandler receives requests from trusted internal services
// (currently: services/payment's Xendit callback relay, FR-5.6).
type InternalHandler struct {
	internalSecret string
	order          *order.Usecase
}

func NewInternalHandler(internalSecret string, order *order.Usecase) *InternalHandler {
	return &InternalHandler{internalSecret: internalSecret, order: order}
}

type paymentCallbackRequest struct {
	CheckoutGroupID string `json:"checkout_group_id"`
	XenditInvoiceID string `json:"xendit_invoice_id"`
	Status          string `json:"status"`
}

// PaymentCallback handles POST /internal/payments/callback, relayed by
// services/payment after it verifies the Xendit webhook.
func (h *InternalHandler) PaymentCallback(c fiber.Ctx) error {
	if c.Get("X-Internal-Secret") != h.internalSecret {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error("Invalid internal secret", nil))
	}

	var req paymentCallbackRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}

	if req.Status == "PAID" {
		if err := h.order.HandlePaidCallback(c.Context(), req.XenditInvoiceID); err != nil {
			return errJSON(c, err)
		}
	}

	return c.JSON(response.Success("Callback received", nil))
}
