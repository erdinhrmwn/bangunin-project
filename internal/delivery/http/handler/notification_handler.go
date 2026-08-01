package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	notificationusecase "erdinhrmwn/bangunin/internal/usecase/notification"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
)

// NotificationHandler serves the caller's own in-app notifications, mounted
// under /user/notifications.
type NotificationHandler struct {
	notify *notificationusecase.Usecase
}

func NewNotificationHandler(notify *notificationusecase.Usecase) *NotificationHandler {
	return &NotificationHandler{notify: notify}
}

func (h *NotificationHandler) List(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	notifs, total, err := h.notify.List(c.Context(), userID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", notifs, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *NotificationHandler) MarkRead(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.notify.MarkRead(c.Context(), id, userID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Notification marked as read", nil))
}
