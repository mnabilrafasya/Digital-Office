package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// minJWTSecretLength adalah panjang minimal JWT_SECRET (karakter)
const minJWTSecretLength = 32

// placeholderPrefix adalah awalan nilai contoh di .env.example yang wajib diganti
const placeholderPrefix = "ganti_dengan"

// menampung semua pengaturan aplikasi dari .env
type Config struct {
	AppEnv  string
	AppPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool

	JWTSecret string
	JWTTTL    time.Duration

	CORSAllowedOrigins []string
}

// membaca file .env
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "production"),
		AppPort: getEnv("APP_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "1433"),
		DBUser:     getEnv("DB_USER", "sa"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "digioffice"),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: os.Getenv("MINIO_ROOT_USER"),
		MinioSecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		MinioUseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",

		JWTSecret: os.Getenv("JWT_SECRET"),

		CORSAllowedOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	if cfg.DBPassword == "" || cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" {
		return nil, errors.New("DB_PASSWORD, MINIO_ROOT_USER, dan MINIO_ROOT_PASSWORD wajib diisi di .env")
	}
	// JWT_TTL yang tidak valid dibiarkan 0 dan baru ditolak oleh ValidateJWT,
	// supaya alat yang tidak memakai token (migrate, seed) tetap bisa berjalan
	if ttl, err := time.ParseDuration(getEnv("JWT_TTL", "8h")); err == nil {
		cfg.JWTTTL = ttl
	}

	return cfg, nil
}

// ValidateJWT memeriksa konfigurasi token. Hanya dipanggil oleh server API.
func (c *Config) ValidateJWT() error {
	if strings.HasPrefix(c.JWTSecret, placeholderPrefix) {
		return errors.New("JWT_SECRET di .env masih berisi nilai contoh dari .env.example, ganti dengan string acak")
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET wajib diisi di .env, minimal %d karakter", minJWTSecretLength)
	}
	if c.JWTTTL <= 0 {
		return errors.New("JWT_TTL tidak valid, contoh yang benar: 8h atau 30m")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitList memecah "a, b,c" menjadi ["a" "b" "c"] dan membuang isian kosong
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
