package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
)

// membuka koneksi ke SQL Server
func NewSQLServer(cfg *config.Config) (*sql.DB, error) {
	query := url.Values{}
	query.Set("database", cfg.DBName)
	query.Set("encrypt", "true")
	query.Set("TrustServerCertificate", "true")

	dsn := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:     fmt.Sprintf("%s:%s", cfg.DBHost, cfg.DBPort),
		RawQuery: query.Encode(),
	}

	db, err := sql.Open("sqlserver", dsn.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("gagal terhubung ke SQL Server: %w", err)
	}
	return db, nil
}