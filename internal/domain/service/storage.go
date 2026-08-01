package service

import (
	"context"
	"io"
)

// StorageService uploads/deletes objects in an S3-compatible bucket
// (MinIO in dev, Cloudflare R2 in prod). Implemented by infra/storage.
type StorageService interface {
	Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) (url string, err error)
	Delete(ctx context.Context, key string) error
}
