package config

import (
	"strings"
	"testing"
	"time"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PASSWORD", "x")
	t.Setenv("MINIO_ROOT_USER", "x")
	t.Setenv("MINIO_ROOT_PASSWORD", "x")
}

// Alat yang tidak memakai token (migrate, seed) harus tetap bisa membaca konfigurasi
func TestLoadToleratesAPIOnlySettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_TTL", "bukan-durasi")
	t.Setenv("UPLOAD_MAX_FILE_SIZE_MB", "abc")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	err = cfg.ValidateAPI()
	if err == nil {
		t.Fatal("ValidateAPI() = nil, want error")
	}
	for _, want := range []string{"JWT_TTL", "UPLOAD_MAX_FILE_SIZE_MB", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q tidak menyebut %s", err, want)
		}
	}
}

func TestDBTrustServerCertDefault(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_TRUST_SERVER_CERTIFICATE", "")

	t.Setenv("APP_ENV", "development")
	if cfg, _ := Load(); !cfg.DBTrustServerCert {
		t.Error("development: DBTrustServerCert = false, want true")
	}
	t.Setenv("APP_ENV", "production")
	if cfg, _ := Load(); cfg.DBTrustServerCert {
		t.Error("production: DBTrustServerCert = true, want false")
	}
}

func TestValidateAPI(t *testing.T) {
	valid := func() *Config {
		return &Config{
			JWTSecret: strings.Repeat("a", 32), JWTTTL: 8 * time.Hour, AppTimezone: "Asia/Jakarta",
			UploadMaxFileSizeMB: 25, UploadURLTTL: 15 * time.Minute, DownloadURLTTL: 5 * time.Minute,
			StorageMinFreeMB: 1024,
		}
	}
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{"valid", func(*Config) {}, false},
		{"placeholder .env.example", func(c *Config) { c.JWTSecret = "ganti_dengan_string_acak_minimal_32_karakter" }, true},
		{"secret terlalu pendek", func(c *Config) { c.JWTSecret = strings.Repeat("a", 31) }, true},
		{"TTL nol", func(c *Config) { c.JWTTTL = 0 }, true},
		{"upload 0 MB", func(c *Config) { c.UploadMaxFileSizeMB = 0 }, true},
		{"upload 1025 MB", func(c *Config) { c.UploadMaxFileSizeMB = 1025 }, true},
		{"upload URL TTL nol", func(c *Config) { c.UploadURLTTL = 0 }, true},
		{"zona waktu salah", func(c *Config) { c.AppTimezone = "Mars/Olympus" }, true},
	}
	for _, tt := range tests {
		cfg := valid()
		tt.mutate(cfg)
		err := cfg.ValidateAPI()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: ValidateAPI() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
		if err == nil && cfg.Location == nil {
			t.Errorf("%s: Location tidak diisi", tt.name)
		}
	}
}
