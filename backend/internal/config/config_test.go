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
func TestLoadWithoutJWTSecret(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_TTL", "bukan-durasi")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if err := cfg.ValidateJWT(); err == nil {
		t.Fatal("ValidateJWT() = nil, want error untuk secret kosong")
	}
}

func TestValidateJWT(t *testing.T) {
	valid := strings.Repeat("a", 32)
	tests := []struct {
		name    string
		secret  string
		ttl     time.Duration
		wantErr bool
	}{
		{"valid", valid, 8 * time.Hour, false},
		{"placeholder .env.example", "ganti_dengan_string_acak_minimal_32_karakter", 8 * time.Hour, true},
		{"terlalu pendek", strings.Repeat("a", 31), 8 * time.Hour, true},
		{"kosong", "", 8 * time.Hour, true},
		{"TTL tidak valid", valid, 0, true},
	}
	for _, tt := range tests {
		cfg := &Config{JWTSecret: tt.secret, JWTTTL: tt.ttl}
		if err := cfg.ValidateJWT(); (err != nil) != tt.wantErr {
			t.Errorf("%s: ValidateJWT() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
