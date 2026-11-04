package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// PutDocument menyimpan file surat (naskah atau lampiran) ke bucket documents dengan nama objectKey.
func (s *Storage) PutDocument(ctx context.Context, objectKey string, data []byte, contentType string) error {
	return s.put(ctx, BucketDocuments, objectKey, data, contentType)
}

// OpenDocument membuka file surat untuk dibaca beserta ukurannya. Pemanggil wajib menutup reader-nya.
func (s *Storage) OpenDocument(ctx context.Context, objectKey string) (io.ReadCloser, int64, error) {
	return s.open(ctx, BucketDocuments, objectKey)
}

// RemoveDocument menghapus file surat (dipakai saat lampiran draf dihapus).
func (s *Storage) RemoveDocument(ctx context.Context, objectKey string) error {
	if err := s.client.RemoveObject(ctx, BucketDocuments, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("menghapus file dari MinIO: %w", err)
	}
	return nil
}

// PutTemplate menyimpan file template surat ke bucket templates dengan nama objectKey.
func (s *Storage) PutTemplate(ctx context.Context, objectKey string, data []byte, contentType string) error {
	return s.put(ctx, BucketTemplates, objectKey, data, contentType)
}

// OpenTemplate membuka file template surat beserta ukurannya. Pemanggil wajib menutup reader-nya.
func (s *Storage) OpenTemplate(ctx context.Context, objectKey string) (io.ReadCloser, int64, error) {
	return s.open(ctx, BucketTemplates, objectKey)
}

func (s *Storage) put(ctx context.Context, bucket, objectKey string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, objectKey, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("menyimpan file ke MinIO (%s): %w", bucket, err)
	}
	return nil
}

func (s *Storage) open(ctx context.Context, bucket, objectKey string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("membuka file dari MinIO (%s): %w", bucket, err)
	}
	info, err := obj.Stat() // GetObject malas; error (mis. objek tidak ada) baru muncul di sini
	if err != nil {
		obj.Close()
		return nil, 0, fmt.Errorf("membuka file dari MinIO (%s): %w", bucket, err)
	}
	return obj, info.Size, nil
}
