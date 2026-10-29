// Package db berisi koneksi SQL Server, helper transaksi, dan pengulangan otomatis saat deadlock.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"time"

	_ "github.com/microsoft/go-mssqldb" // driver "sqlserver"

	"office.backend/internal/platform/config"
)

// BuildDSN menyusun alamat koneksi SQL Server ke database tertentu.
// dbName dibuat parameter supaya alat migration bisa terhubung ke "master"
// (untuk membuat database) maupun ke database aplikasi.
func BuildDSN(cfg *config.Config, dbName string) string {
	query := url.Values{}
	query.Set("database", dbName)
	query.Set("encrypt", "true")
	query.Set("TrustServerCertificate", strconv.FormatBool(cfg.DBTrustServerCert))
	query.Set("app name", "digioffice")

	dsn := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:     fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
		RawQuery: query.Encode(),
	}
	return dsn.String()
}

// Open membuka pool koneksi ke database aplikasi dan memastikan server bisa dihubungi.
func Open(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	conn, err := sql.Open("sqlserver", BuildDSN(cfg, cfg.DBName))
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi SQL Server: %w", err)
	}
	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("gagal terhubung ke SQL Server: %w", err)
	}
	return conn, nil
}
