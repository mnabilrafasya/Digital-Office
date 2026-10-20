package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// PresignedUpload adalah izin unggah langsung ke MinIO dengan POST multipart.
type PresignedUpload struct {
	URL       string
	Fields    map[string]string
	ExpiresAt time.Time
}

// PresignUpload membuat presigned POST untuk object key, dengan batas ukuran 1..maxSize byte
// yang ditegakkan oleh MinIO sendiri (content-length-range).
func (s *Storage) PresignUpload(ctx context.Context, key string, maxSize int64, ttl time.Duration) (PresignedUpload, error) {
	expiresAt := time.Now().Add(ttl)
	policy := minio.NewPostPolicy()
	if err := policy.SetBucket(s.bucket); err != nil {
		return PresignedUpload{}, fmt.Errorf("menyusun kebijakan unggah: %w", err)
	}
	if err := policy.SetKey(key); err != nil {
		return PresignedUpload{}, fmt.Errorf("menyusun kebijakan unggah: %w", err)
	}
	if err := policy.SetExpires(expiresAt); err != nil {
		return PresignedUpload{}, fmt.Errorf("menyusun kebijakan unggah: %w", err)
	}
	if err := policy.SetContentLengthRange(1, maxSize); err != nil {
		return PresignedUpload{}, fmt.Errorf("menyusun kebijakan unggah: %w", err)
	}
	// Browser boleh mengirim field Content-Type berisi apa pun (tidak ada pengecekan jenis file)
	if err := policy.SetContentTypeStartsWith(""); err != nil {
		return PresignedUpload{}, fmt.Errorf("menyusun kebijakan unggah: %w", err)
	}

	u, fields, err := s.presign.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return PresignedUpload{}, fmt.Errorf("membuat presigned POST: %w", err)
	}
	return PresignedUpload{URL: u.String(), Fields: fields, ExpiresAt: expiresAt}, nil
}

// PresignDownload membuat presigned GET yang selalu membuat browser MENGUNDUH file (bukan membuka
// atau menjalankannya): Content-Disposition attachment dan Content-Type application/octet-stream.
func (s *Storage) PresignDownload(ctx context.Context, key, fileName string, ttl time.Duration) (string, time.Time, error) {
	params := url.Values{}
	params.Set("response-content-disposition", ContentDisposition(fileName))
	params.Set("response-content-type", "application/octet-stream")

	expiresAt := time.Now().Add(ttl)
	u, err := s.presign.PresignedGetObject(ctx, s.bucket, key, ttl, params)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("membuat presigned GET: %w", err)
	}
	return u.String(), expiresAt, nil
}

// ContentDisposition membuat header unduhan berisi nama file ASCII sebagai cadangan (parameter
// filename) dan nama UTF-8 asli yang di-encode persen (parameter filename*, RFC 6266 / RFC 5987).
func ContentDisposition(fileName string) string {
	var ascii strings.Builder
	for _, r := range fileName {
		switch {
		case r < 0x20 || r > 0x7e || r == '"' || r == '\\':
			ascii.WriteByte('_')
		default:
			ascii.WriteRune(r)
		}
	}
	encoded := strings.ReplaceAll(url.PathEscape(fileName), "+", "%2B")
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii.String(), encoded)
}
