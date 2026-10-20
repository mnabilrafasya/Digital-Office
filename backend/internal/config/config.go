// Package config membaca pengaturan aplikasi dari environment (dan file .env bila ada).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	// minJWTSecretLength adalah panjang minimal JWT_SECRET (karakter)
	minJWTSecretLength = 32
	// placeholderPrefix adalah awalan nilai contoh di .env.example yang wajib diganti
	placeholderPrefix = "ganti_dengan"
	// EnvDevelopment adalah nilai APP_ENV untuk komputer developer
	EnvDevelopment = "development"

	maxUploadSizeMB = 1024
)

// Config menampung semua pengaturan aplikasi.
type Config struct {
	AppEnv            string
	AppPort           string
	AppTimezone       string
	Location          *time.Location // diisi oleh ValidateAPI
	MemoryLimitMB     int
	TrustProxyHeaders bool

	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBTrustServerCert bool

	MinioEndpoint       string
	MinioAccessKey      string
	MinioSecretKey      string
	MinioUseSSL         bool
	MinioPublicEndpoint string
	MinioPublicUseSSL   bool
	MinioRegion         string

	JWTSecret string
	JWTTTL    time.Duration

	CORSAllowedOrigins []string

	UploadMaxFileSizeMB int
	UploadURLTTL        time.Duration
	DownloadURLTTL      time.Duration
	StorageMinFreeMB    int

	// invalid berisi pengaturan yang formatnya salah. Tidak menggagalkan Load (supaya migrate
	// dan seed tetap jalan), tetapi ditolak oleh ValidateAPI.
	invalid []string
}

// Load membaca file .env (bila ada) lalu environment. Hanya pengaturan yang dibutuhkan
// semua alat (database dan storage) yang diwajibkan di sini.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:      getEnv("APP_ENV", "production"),
		AppPort:     getEnv("APP_PORT", "8080"),
		AppTimezone: getEnv("APP_TIMEZONE", "Asia/Jakarta"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "1433"),
		DBUser:     getEnv("DB_USER", "sa"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnv("DB_NAME", "digioffice"),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: os.Getenv("MINIO_ROOT_USER"),
		MinioSecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		MinioRegion:    getEnv("MINIO_REGION", "us-east-1"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		CORSAllowedOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	// Sertifikat SQL Server hanya boleh tidak diperiksa di komputer developer
	cfg.DBTrustServerCert = cfg.parseBool("DB_TRUST_SERVER_CERTIFICATE", cfg.IsDevelopment())
	cfg.TrustProxyHeaders = cfg.parseBool("TRUST_PROXY_HEADERS", false)
	cfg.MinioUseSSL = cfg.parseBool("MINIO_USE_SSL", false)
	cfg.MinioPublicEndpoint = getEnv("MINIO_PUBLIC_ENDPOINT", cfg.MinioEndpoint)
	cfg.MinioPublicUseSSL = cfg.parseBool("MINIO_PUBLIC_USE_SSL", cfg.MinioUseSSL)

	cfg.MemoryLimitMB = cfg.parseInt("APP_MEMORY_LIMIT_MB", 0)
	cfg.UploadMaxFileSizeMB = cfg.parseInt("UPLOAD_MAX_FILE_SIZE_MB", 25)
	cfg.StorageMinFreeMB = cfg.parseInt("STORAGE_MIN_FREE_MB", 1024)

	cfg.JWTTTL = cfg.parseDuration("JWT_TTL", 8*time.Hour)
	cfg.UploadURLTTL = cfg.parseDuration("UPLOAD_URL_TTL", 15*time.Minute)
	cfg.DownloadURLTTL = cfg.parseDuration("DOWNLOAD_URL_TTL", 5*time.Minute)

	if cfg.DBPassword == "" || cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" {
		return nil, errors.New("DB_PASSWORD, MINIO_ROOT_USER, dan MINIO_ROOT_PASSWORD wajib diisi di .env")
	}
	return cfg, nil
}

// IsDevelopment mengembalikan true bila APP_ENV=development.
func (c *Config) IsDevelopment() bool { return c.AppEnv == EnvDevelopment }

// UploadMaxBytes adalah batas ukuran satu file unggahan dalam byte.
func (c *Config) UploadMaxBytes() int64 { return int64(c.UploadMaxFileSizeMB) << 20 }

// StorageMinFreeBytes adalah sisa ruang storage minimal yang harus dijaga, dalam byte.
func (c *Config) StorageMinFreeBytes() int64 { return int64(c.StorageMinFreeMB) << 20 }

// ValidateAPI memeriksa pengaturan yang hanya dibutuhkan server API (token, upload, zona waktu).
// Dipanggil saat startup; seluruh masalah dilaporkan sekaligus.
func (c *Config) ValidateAPI() error {
	var errs []error
	for _, key := range c.invalid {
		errs = append(errs, fmt.Errorf("%s formatnya tidak valid", key))
	}

	switch {
	case strings.HasPrefix(c.JWTSecret, placeholderPrefix):
		errs = append(errs, errors.New("JWT_SECRET di .env masih berisi nilai contoh dari .env.example, ganti dengan string acak"))
	case len(c.JWTSecret) < minJWTSecretLength:
		errs = append(errs, fmt.Errorf("JWT_SECRET wajib diisi di .env, minimal %d karakter", minJWTSecretLength))
	}
	if c.JWTTTL <= 0 {
		errs = append(errs, errors.New("JWT_TTL harus lebih dari 0, contoh: 8h"))
	}
	if c.UploadMaxFileSizeMB < 1 || c.UploadMaxFileSizeMB > maxUploadSizeMB {
		errs = append(errs, fmt.Errorf("UPLOAD_MAX_FILE_SIZE_MB harus antara 1 dan %d", maxUploadSizeMB))
	}
	if c.UploadURLTTL <= 0 || c.DownloadURLTTL <= 0 {
		errs = append(errs, errors.New("UPLOAD_URL_TTL dan DOWNLOAD_URL_TTL harus lebih dari 0, contoh: 15m"))
	}
	if c.StorageMinFreeMB < 0 || c.MemoryLimitMB < 0 {
		errs = append(errs, errors.New("STORAGE_MIN_FREE_MB dan APP_MEMORY_LIMIT_MB tidak boleh negatif"))
	}

	loc, err := time.LoadLocation(c.AppTimezone)
	if err != nil {
		errs = append(errs, fmt.Errorf("APP_TIMEZONE %q tidak dikenal", c.AppTimezone))
	}
	c.Location = loc

	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (c *Config) parseBool(key string, fallback bool) bool {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		c.invalid = append(c.invalid, key)
		return fallback
	}
	return v
}

func (c *Config) parseInt(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		c.invalid = append(c.invalid, key)
		return fallback
	}
	return v
}

func (c *Config) parseDuration(key string, fallback time.Duration) time.Duration {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		c.invalid = append(c.invalid, key)
		return 0
	}
	return v
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
