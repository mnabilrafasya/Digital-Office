// Package storage berisi akses ke MinIO (object storage) dan penyiapan bucket aplikasi.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"office.backend/internal/platform/config"
)

// Bucket aplikasi.
const (
	BucketDocuments = "documents"
	BucketTemplates = "templates"
)

// Storage membungkus client MinIO yang dipakai aplikasi.
type Storage struct {
	client *minio.Client
}

// New membuat koneksi ke MinIO dan memastikan bucket aplikasi sudah ada.
func New(ctx context.Context, cfg *config.Config) (*Storage, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("membuat client MinIO: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for _, bucket := range []string{BucketDocuments, BucketTemplates} {
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			return nil, fmt.Errorf("gagal terhubung ke MinIO: %w", err)
		}
		if !exists {
			if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return nil, fmt.Errorf("gagal membuat bucket %s: %w", bucket, err)
			}
		}
	}
	return &Storage{client: client}, nil
}

// Ping memastikan MinIO bisa dihubungi (dipakai /health).
func (s *Storage) Ping(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx)
	return err
}
