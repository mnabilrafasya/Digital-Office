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
	t.Setenv("APP_MEMORY_LIMIT_MB", "abc")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	err = cfg.ValidateAPI()
	if err == nil {
		t.Fatal("ValidateAPI() = nil, want error")
	}
	for _, want := range []string{"JWT_TTL", "APP_MEMORY_LIMIT_MB", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q tidak menyebut %s", err, want)
		}
	}
}

func TestLoadDefaults(t *testing.T) {
	setRequiredEnv(t)
	for _, k := range []string{"DB_TRUST_SERVER_CERTIFICATE", "TRUST_PROXY_HEADERS", "APP_TIMEZONE", "APP_MEMORY_LIMIT_MB"} {
		t.Setenv(k, "")
	}

	t.Setenv("APP_ENV", "development")
	cfg, _ := Load()
	if !cfg.DBTrustServerCert || cfg.TrustProxyHeaders || cfg.AppTimezone != "Asia/Jakarta" || cfg.MemoryLimitMB != 0 {
		t.Errorf("default development salah: %+v", cfg)
	}
	t.Setenv("APP_ENV", "production")
	if cfg, _ := Load(); cfg.DBTrustServerCert {
		t.Error("production: DBTrustServerCert = true, want false")
	}
}

func TestValidateAPI(t *testing.T) {
	valid := func() *Config {
		return &Config{JWTSecret: strings.Repeat("a", 32), JWTTTL: 8 * time.Hour, AppTimezone: "Asia/Jakarta"}
	}
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{"valid", func(*Config) {}, false},
		{"placeholder .env.example", func(c *Config) { c.JWTSecret = "ganti_dengan_string_acak_minimal_32_karakter" }, true},
		{"secret terlalu pendek", func(c *Config) { c.JWTSecret = strings.Repeat("a", 31) }, true},
		{"secret kosong", func(c *Config) { c.JWTSecret = "" }, true},
		{"TTL nol", func(c *Config) { c.JWTTTL = 0 }, true},
		{"batas memori negatif", func(c *Config) { c.MemoryLimitMB = -1 }, true},
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
