package service

import (
	"context"
	"io"
	"time"
)

// StorageService uploads/deletes objects in an S3-compatible bucket
// (MinIO in dev, Cloudflare R2 in prod). Implemented by infra/storage.
type StorageService interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) (url string, err error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignedURL(ctx context.Context, key string, expiry time.Duration) (url string, err error)
}
