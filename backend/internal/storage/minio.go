// Package storage berisi akses ke MinIO: bucket, presigned URL, operasi object, dan cek kapasitas.
// Byte file tidak pernah lewat server Go; browser mengunggah dan mengunduh langsung ke MinIO.
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/model"
)

// Bucket aplikasi.
const (
	BucketDocuments = "documents"
	BucketTemplates = "templates"
)

// stagingRuleID adalah ID aturan lifecycle yang menghapus unggahan staging yang ditinggal.
const stagingRuleID = "digioffice-staging-expiry"

// ErrObjectNotFound dikembalikan bila object tidak ada di storage.
var ErrObjectNotFound = errors.New("object tidak ditemukan di storage")

// Storage membungkus dua client MinIO: client untuk operasi internal (jaringan server) dan
// client presign yang memakai endpoint publik (alamat yang bisa dijangkau browser). Keduanya
// diberi region supaya pembuatan presigned URL tidak memanggil jaringan.
type Storage struct {
	client  *minio.Client
	presign *minio.Client
	bucket  string
}

// New membuat koneksi MinIO, memastikan bucket ada, dan memasang lifecycle staging (idempoten).
func New(ctx context.Context, cfg *config.Config) (*Storage, error) {
	creds := credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, "")
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{Creds: creds, Secure: cfg.MinioUseSSL, Region: cfg.MinioRegion})
	if err != nil {
		return nil, fmt.Errorf("membuat client MinIO: %w", err)
	}
	presign, err := minio.New(cfg.MinioPublicEndpoint, &minio.Options{Creds: creds, Secure: cfg.MinioPublicUseSSL, Region: cfg.MinioRegion})
	if err != nil {
		return nil, fmt.Errorf("membuat client presign MinIO: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for _, bucket := range []string{BucketDocuments, BucketTemplates} {
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			return nil, fmt.Errorf("gagal terhubung ke MinIO: %w", err)
		}
		if !exists {
			if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: cfg.MinioRegion}); err != nil {
				return nil, fmt.Errorf("gagal membuat bucket %s: %w", bucket, err)
			}
		}
	}

	s := &Storage{client: client, presign: presign, bucket: BucketDocuments}
	if err := s.ensureStagingLifecycle(ctx); err != nil {
		log.Printf("PERINGATAN: lifecycle staging belum terpasang, unggahan yang ditinggal tidak terhapus otomatis: %v", err)
	}
	return s, nil
}

// Ping memastikan MinIO bisa dihubungi (dipakai /health).
func (s *Storage) Ping(ctx context.Context) error {
	_, err := s.client.ListBuckets(ctx)
	return err
}

// ensureStagingLifecycle memasang aturan "hapus object staging/ setelah 1 hari" tanpa
// mengganggu aturan lain yang sudah ada di bucket.
func (s *Storage) ensureStagingLifecycle(ctx context.Context) error {
	cfg, err := s.client.GetBucketLifecycle(ctx, s.bucket)
	if err != nil {
		if minio.ToErrorResponse(err).Code != "NoSuchLifecycleConfiguration" {
			return fmt.Errorf("membaca lifecycle: %w", err)
		}
		cfg = lifecycle.NewConfiguration()
	}

	rule := lifecycle.Rule{
		ID:         stagingRuleID,
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: stagingPrefix},
		Expiration: lifecycle.Expiration{Days: 1},
	}
	rules := cfg.Rules[:0]
	for _, r := range cfg.Rules {
		if r.ID == stagingRuleID {
			if r.Status == rule.Status && r.RuleFilter.Prefix == stagingPrefix && r.Expiration.Days == 1 {
				return nil // sudah terpasang dengan benar
			}
			continue
		}
		rules = append(rules, r)
	}
	cfg.Rules = append(rules, rule)

	if err := s.client.SetBucketLifecycle(ctx, s.bucket, cfg); err != nil {
		return fmt.Errorf("memasang lifecycle: %w", err)
	}
	return nil
}

// Stat mengambil ukuran dan content type object.
func (s *Storage) Stat(ctx context.Context, key string) (model.StoredObject, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return model.StoredObject{}, ErrObjectNotFound
		}
		return model.StoredObject{}, fmt.Errorf("memeriksa object: %w", err)
	}
	return model.StoredObject{Size: info.Size, ContentType: info.ContentType}, nil
}

// Copy menyalin object di sisi server MinIO (byte tidak lewat Go).
func (s *Storage) Copy(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx,
		minio.CopyDestOptions{Bucket: s.bucket, Object: dstKey},
		minio.CopySrcOptions{Bucket: s.bucket, Object: srcKey})
	if err != nil {
		if isNotFound(err) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("menyalin object: %w", err)
	}
	return nil
}

// Checksum menghitung SHA-256 object secara streaming (memori konstan berapa pun ukuran file).
func (s *Storage) Checksum(ctx context.Context, key string) (string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("membuka object: %w", err)
	}
	defer obj.Close()

	h := sha256.New()
	if _, err := io.Copy(h, obj); err != nil {
		if isNotFound(err) {
			return "", ErrObjectNotFound
		}
		return "", fmt.Errorf("membaca object: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Remove menghapus object (tidak error bila object sudah tidak ada).
func (s *Storage) Remove(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil && !isNotFound(err) {
		return fmt.Errorf("menghapus object: %w", err)
	}
	return nil
}

func isNotFound(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.StatusCode == http.StatusNotFound || resp.Code == "NoSuchKey"
}
