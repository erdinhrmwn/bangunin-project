package handler

import (
	"github.com/gofiber/fiber/v3"

	mediausecase "erdinhrmwn/bangunin/internal/usecase/media"
	"erdinhrmwn/bangunin/pkg/response"
)

// MediaHandler serves POST /media/upload — generic file upload used by
// supplier onboarding documents (FR-3.1).
type MediaHandler struct {
	media *mediausecase.Usecase
}

func NewMediaHandler(media *mediausecase.Usecase) *MediaHandler {
	return &MediaHandler{media: media}
}

func (h *MediaHandler) Upload(c fiber.Ctx) error {
	scope := c.FormValue("scope")
	if scope == "" {
		return badRequest(c)
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return badRequest(c)
	}

	f, err := fh.Open()
	if err != nil {
		return badRequest(c)
	}
	defer f.Close()

	res, err := h.media.Upload(c.Context(), scope, fh.Header.Get("Content-Type"), fh.Size, f)
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("File uploaded", res))
}
