// Package storage implements the FileStorage port using MinIO/S3.
package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

// MinIOAdapter implements port.FileStorage using MinIO/S3-compatible object storage.
type MinIOAdapter struct {
	client *minio.Client
	bucket string
	logger *slog.Logger
}

// NewMinIOAdapter creates a new MinIO storage adapter.
func NewMinIOAdapter(client *minio.Client, bucket string, logger *slog.Logger) *MinIOAdapter {
	return &MinIOAdapter{
		client: client,
		bucket: bucket,
		logger: logger.With("adapter", "minio"),
	}
}

// Upload stores the contents of reader at the given key.
func (a *MinIOAdapter) Upload(ctx context.Context, key string, reader io.Reader, size int64) (string, error) {
	// Ensure bucket exists
	exists, err := a.client.BucketExists(ctx, a.bucket)
	if err != nil {
		return "", fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := a.client.MakeBucket(ctx, a.bucket, minio.MakeBucketOptions{}); err != nil {
			return "", fmt.Errorf("create bucket: %w", err)
		}
	}

	opts := minio.PutObjectOptions{}
	if size <= 0 {
		opts.PartSize = 10 * 1024 * 1024 // 10MB parts for unknown size
		size = -1
	}

	info, err := a.client.PutObject(ctx, a.bucket, key, reader, size, opts)
	if err != nil {
		return "", fmt.Errorf("put object %s: %w", key, err)
	}

	storagePath := fmt.Sprintf("%s/%s", a.bucket, info.Key)
	a.logger.Debug("file uploaded", "key", key, "size", info.Size, "path", storagePath)
	return storagePath, nil
}

// Delete removes the object at the given key.
func (a *MinIOAdapter) Delete(ctx context.Context, key string) error {
	err := a.client.RemoveObject(ctx, a.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object %s: %w", key, err)
	}
	return nil
}

// DeletePrefix removes all objects matching the prefix.
func (a *MinIOAdapter) DeletePrefix(ctx context.Context, prefix string) error {
	objCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objCh {
		if obj.Err != nil {
			return fmt.Errorf("list objects: %w", obj.Err)
		}
		if err := a.client.RemoveObject(ctx, a.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			a.logger.Error("failed to remove object", "key", obj.Key, "error", err)
		}
	}

	a.logger.Info("prefix deleted", "prefix", prefix)
	return nil
}
