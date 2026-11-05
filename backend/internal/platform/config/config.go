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
	// EnvDevelopment adalah nilai APP_ENV untuk komputer developer.
	EnvDevelopment = "development"
)

// Config menampung semua pengaturan aplikasi dari .env.
type Config struct {
	AppEnv            string
	AppPort           string
	AppTimezone       string
	Location          *time.Location // diisi oleh ValidateAPI
	MemoryLimitMB     int            // 0 = tanpa batas
	TrustProxyHeaders bool

	// Antrean unggahan: jatah memori total semua unggahan, batas menunggu giliran, dan batas waktu
	// khusus request unggahan (mengirim file dan total).
	UploadMemoryMB    int
	UploadWait        time.Duration
	UploadReadTimeout time.Duration
	UploadTimeout     time.Duration

	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBTrustServerCert bool // false = sertifikat TLS SQL Server diverifikasi

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool

	GotenbergURL string // layanan LibreOffice (Gotenberg) untuk membuat PDF resmi

	HCOrganizationURL string // API HC PUSRI daftar unit (hanya untuk cmd/sync-hc), rahasia: hanya di .env
	HCToken           string

	JWTSecret string
	JWTTTL    time.Duration

	CORSAllowedOrigins []string

	// invalid berisi pengaturan yang formatnya salah. Tidak menggagalkan Load (supaya migrate
	// dan seed tetap jalan), tetapi ditolak oleh ValidateAPI.
	invalid []string
}

// Load membaca file .env (bila ada) lalu environment. Hanya pengaturan yang dibutuhkan semua
// alat (database dan storage) yang diwajibkan di sini.
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

		GotenbergURL: getEnv("GOTENBERG_URL", "http://localhost:3001"),

		HCOrganizationURL: os.Getenv("PUSRI_HC_ORGANIZATION_URL"),
		HCToken:           os.Getenv("PUSRI_API_TOKEN"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		CORSAllowedOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	// Sertifikat SQL Server hanya boleh tidak diperiksa di komputer developer
	cfg.DBTrustServerCert = cfg.parseBool("DB_TRUST_SERVER_CERTIFICATE", cfg.IsDevelopment())
	cfg.MinioUseSSL = cfg.parseBool("MINIO_USE_SSL", false)
	cfg.TrustProxyHeaders = cfg.parseBool("TRUST_PROXY_HEADERS", false)
	cfg.MemoryLimitMB = cfg.parseInt("APP_MEMORY_LIMIT_MB", 0)
	cfg.JWTTTL = cfg.parseDuration("JWT_TTL", 8*time.Hour)
	cfg.UploadMemoryMB = cfg.parseInt("UPLOAD_MEMORY_MB", 200)
	cfg.UploadWait = cfg.parseDuration("UPLOAD_WAIT", 60*time.Second)
	cfg.UploadReadTimeout = cfg.parseDuration("UPLOAD_READ_TIMEOUT", 2*time.Minute)
	cfg.UploadTimeout = cfg.parseDuration("UPLOAD_TIMEOUT", 3*time.Minute)

	if cfg.DBPassword == "" || cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" {
		return nil, errors.New("DB_PASSWORD, MINIO_ROOT_USER, dan MINIO_ROOT_PASSWORD wajib diisi di .env")
	}
	return cfg, nil
}

// IsDevelopment mengembalikan true bila APP_ENV=development.
func (c *Config) IsDevelopment() bool { return c.AppEnv == EnvDevelopment }

// ValidateAPI memeriksa pengaturan yang hanya dibutuhkan server API (token, zona waktu, memori).
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
		errs = append(errs, errors.New("JWT_TTL harus lebih dari 0, contoh: 8h atau 30m"))
	}
	if c.MemoryLimitMB < 0 {
		errs = append(errs, errors.New("APP_MEMORY_LIMIT_MB tidak boleh negatif (0 = tanpa batas)"))
	}
	if c.UploadMemoryMB < 1 {
		errs = append(errs, errors.New("UPLOAD_MEMORY_MB harus lebih dari 0"))
	}
	if c.UploadWait <= 0 || c.UploadReadTimeout <= 0 || c.UploadTimeout <= 0 {
		errs = append(errs, errors.New("UPLOAD_WAIT, UPLOAD_READ_TIMEOUT, dan UPLOAD_TIMEOUT harus lebih dari 0, contoh: 60s atau 2m"))
	} else if c.UploadTimeout <= c.UploadWait || c.UploadTimeout < c.UploadReadTimeout {
		errs = append(errs, errors.New("UPLOAD_TIMEOUT harus lebih lama dari UPLOAD_WAIT dan tidak kurang dari UPLOAD_READ_TIMEOUT"))
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

// parseDuration mengembalikan 0 bila formatnya salah, supaya ValidateAPI menolaknya.
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
