package handler

import (
	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/pkg/response"
)

// InternalHandler receives requests from trusted internal services
// (currently: services/payment's Xendit callback relay, FR-5.6).
type InternalHandler struct {
	internalSecret string
}

func NewInternalHandler(internalSecret string) *InternalHandler {
	return &InternalHandler{internalSecret: internalSecret}
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

	// TODO(sub-task 10): wire to usecase/order.Usecase.HandlePaidCallback
	// (idempotent on xendit_invoice_id+status) once the order usecase lands.
	_ = req

	return c.JSON(response.Success("Callback received", nil))
}
