// Pengisi data dummy untuk development.
// Jalankan dari folder backend:
//
//	go run ./cmd/seed
//
// Syarat: APP_ENV=development dan SEED_PASSWORD terisi di .env,
// serta migration sudah dijalankan (go run ./cmd/migrate up).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mnabilrafasya/Digital-Office/backend/internal/config"
	"github.com/mnabilrafasya/Digital-Office/backend/internal/database"
)

const (
	seedFile   = "seeds/dev_seed.sql"
	bcryptCost = 12
)

func main() {
	log.SetFlags(0)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Gagal membaca .env: %v", err)
	}
	if cfg.AppEnv != "development" {
		log.Fatalf("Ditolak: data dummy hanya boleh diisi saat APP_ENV=development (sekarang %q).", cfg.AppEnv)
	}

	password := os.Getenv("SEED_PASSWORD")
	if len(password) < 8 {
		log.Fatal("SEED_PASSWORD di .env wajib diisi, minimal 8 karakter.")
	}

	sqlText, err := os.ReadFile(seedFile)
	if err != nil {
		log.Fatalf("Gagal membaca %s (jalankan dari folder backend): %v", seedFile, err)
	}

	db, err := database.NewSQLServer(cfg)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Seed hanya untuk database kosong, supaya tidak ada data ganda
	var existing int
	if err := db.QueryRowContext(ctx,
		"SELECT (SELECT COUNT(*) FROM companies) + (SELECT COUNT(*) FROM roles) + (SELECT COUNT(*) FROM lookups)",
	).Scan(&existing); err != nil {
		log.Fatalf("Gagal memeriksa data (sudah menjalankan go run ./cmd/migrate up?): %v", err)
	}
	if existing > 0 {
		fmt.Println("Data sudah ada, seed dilewati.")
		fmt.Println("Untuk mengulang dari nol (development saja):")
		fmt.Println("  go run ./cmd/migrate down 18")
		fmt.Println("  go run ./cmd/migrate up")
		fmt.Println("  go run ./cmd/seed")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		log.Fatalf("Gagal membuat hash password: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("Gagal memulai transaksi: %v", err)
	}
	if _, err := tx.ExecContext(ctx, string(sqlText), sql.Named("password_hash", string(hash))); err != nil {
		_ = tx.Rollback()
		log.Fatalf("Seed gagal, semua perubahan dibatalkan: %v", err)
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("Gagal menyimpan seed: %v", err)
	}

	fmt.Println("Data dummy berhasil diisi: 2 perusahaan, 13 unit, 22 jabatan, 3 role, 12 permission, 24 user, 56 lookup.")
	fmt.Println("Semua user memakai password dari SEED_PASSWORD di .env.")
}
