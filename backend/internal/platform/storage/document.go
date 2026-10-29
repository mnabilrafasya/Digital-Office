package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// PutDocument menyimpan file naskah surat ke bucket documents dengan nama objectKey.
func (s *Storage) PutDocument(ctx context.Context, objectKey string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, BucketDocuments, objectKey, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("menyimpan dokumen ke MinIO: %w", err)
	}
	return nil
}

// OpenDocument membuka file naskah surat untuk dibaca beserta ukurannya. Pemanggil wajib
// menutup reader-nya.
func (s *Storage) OpenDocument(ctx context.Context, objectKey string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, BucketDocuments, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("membuka dokumen dari MinIO: %w", err)
	}
	info, err := obj.Stat() // GetObject malas; error (mis. objek tidak ada) baru muncul di sini
	if err != nil {
		obj.Close()
		return nil, 0, fmt.Errorf("membuka dokumen dari MinIO: %w", err)
	}
	return obj, info.Size, nil
}
