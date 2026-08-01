// Package media implements generic file upload used by supplier
// onboarding documents and, later, product images (FR-3.1).
package media

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/pkg/apperr"
)

const (
	maxImageSize = 5 << 20  // 5MB
	maxPDFSize   = 10 << 20 // 10MB
)

var extByContentType = map[string]string{
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

func isImage(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp"
}

type Usecase struct {
	storage service.StorageService
	media   service.MediaEnqueuer
}

func New(storage service.StorageService, media service.MediaEnqueuer) *Usecase {
	return &Usecase{storage: storage, media: media}
}

type UploadResult struct {
	URL string
	Key string
}

// Upload validates type/size, stores the object under {scope}/{uuid}.{ext},
// and enqueues a resize job for images (FR-3.1).
func (u *Usecase) Upload(ctx context.Context, scope, contentType string, size int64, r io.Reader) (*UploadResult, error) {
	ext, ok := extByContentType[contentType]
	if !ok {
		return nil, apperr.New("UNSUPPORTED_MEDIA_TYPE", "File type not allowed", 422)
	}

	max := int64(maxPDFSize)
	if isImage(contentType) {
		max = maxImageSize
	}
	if size > max {
		return nil, apperr.New("FILE_TOO_LARGE", "File exceeds the allowed size", 422)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("media: generate id: %w", err)
	}
	key := fmt.Sprintf("%s/%s.%s", scope, id, ext)

	url, err := u.storage.Upload(ctx, key, r, size, contentType)
	if err != nil {
		return nil, err
	}

	if isImage(contentType) {
		if err := u.media.EnqueueMediaProcess(key); err != nil {
			return nil, err
		}
	}

	return &UploadResult{URL: url, Key: key}, nil
}
