// Package storage implements domain/service.StorageService against an
// S3-compatible bucket (MinIO in dev, Cloudflare R2 in prod) via minio-go —
// the same client works against both through a custom endpoint.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"erdinhrmwn/bangunin/config"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
	scheme string
	host   string
}

// New creates the client and ensures the bucket exists (auto-create in dev;
// in prod the R2 bucket is expected to already exist, MakeBucket is a no-op
// if it does).
func New(ctx context.Context, cfg config.R2Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: create client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: check bucket: %w", err)
	}
	if !exists {
		// ponytail: api/worker start concurrently and race on first-boot bucket
		// creation; the loser gets BucketAlreadyOwnedByYou, which isn't a real error.
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			code := minio.ToErrorResponse(err).Code
			if code != "BucketAlreadyOwnedByYou" && code != "BucketAlreadyExists" {
				return nil, fmt.Errorf("storage: create bucket: %w", err)
			}
		}
	}

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	return &MinIOStorage{client: client, bucket: cfg.Bucket, scheme: scheme, host: cfg.Endpoint}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("storage: upload %s: %w", key, err)
	}
	return fmt.Sprintf("%s://%s/%s/%s", s.scheme, s.host, s.bucket, key), nil
}

func (s *MinIOStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage: download %s: %w", key, err)
	}
	return obj, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}
